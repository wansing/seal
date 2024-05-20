package seal

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/mattn/go-isatty"
)

// GitReloadHandler returns a rate-limited handler which runs "git fetch" and "git reset --hard" in osDir, then calls reload.
//
// We can't distinguish between local commits (which should be kept) and upstream history rewrites (which can be dropped).
// Thus it fails if there are local changes and refuses to run from an interactive terminal.
// You should know about "git reflog".
func GitReloadHandler(secret string, osDir string, reload func() error) http.HandlerFunc {
	if osDir == "" {
		return http.NotFound
	}

	limitedReload := Limit(time.Minute, 2, func() error {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			return errors.New("git reload has no effect when running in a terminal")
		}

		status := exec.Command("git", "status", "--porcelain")
		status.Dir = osDir
		localChanges, err := status.Output()
		if err != nil {
			return errors.New("error running git status")
		}
		if len(localChanges) > 0 {
			return errors.New("git working copy has local changes")
		}

		// https://stackoverflow.com/questions/9813816/git-pull-after-forced-update
		// this drops locals commits, however they can be restored with "git reflog" for a while
		fetch := exec.Command("git", "fetch")
		fetch.Dir = osDir
		if err := fetch.Run(); err != nil {
			return fmt.Errorf("error running git fetch: %v", err)
		}
		reset := exec.Command("git", "reset", "--hard", "origin")
		reset.Dir = osDir
		if err := reset.Run(); err != nil {
			return fmt.Errorf("error running git reset: %v", err)
		}

		return reload()
	})

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}

		start := time.Now()
		if done, err := limitedReload(); done {
			if err == nil {
				w.Write([]byte(fmt.Sprintf("git reload took %d milliseconds", time.Since(start).Milliseconds())))
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("git reload failed: %v", err)))
			}
		} else {
			if err == nil {
				w.Write([]byte("git reload scheduled"))
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("git reload scheduled, last execution returned error: %v", err)))
			}
		}
	}
}

// ReloadHandler returns a rate-limited handler which calls reload.
func ReloadHandler(secret string, reload func() error) http.HandlerFunc {
	limitedReload := Limit(time.Minute, 2, func() error {
		return reload()
	})

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}

		start := time.Now()
		if done, err := limitedReload(); done {
			if err == nil {
				w.Write([]byte(fmt.Sprintf("reload took %d milliseconds", time.Since(start).Milliseconds())))
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("reload failed: %v", err)))
			}
		} else {
			if err == nil {
				w.Write([]byte("reload scheduled"))
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("reload scheduled, last execution returned error: %v", err)))
			}
		}
	}
}
