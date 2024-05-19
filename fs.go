package seal

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/mattn/go-isatty"
)

type FS struct {
	Fsys   fs.FS
	GitDir string // for git reload, optional

	root *Dir // set by Reload
}

func DirFS(dir string) *FS {
	return &FS{
		Fsys:   os.DirFS(dir),
		GitDir: dir,
	}
}

// Serve processes the given reqpath, calling the handler of each directory it passes by, until one handler returns false or the path is done.
//
// Serve always returns false. The return value exists for compatibility with type Handler.
func (fs *FS) Serve(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
	dir := fs.root
	for {
		if dir.Handler != nil {
			cont := dir.Handler(reqpath, w, r)
			if !cont {
				return false
			}
		}

		if len(reqpath) == 0 {
			http.NotFound(w, r)
			return false
		}

		next, ok := dir.Subdirs[reqpath[0]]
		if ok {
			reqpath = reqpath[1:]
			dir = next
			continue
		}

		// no subdir with that name found, now try as a file
		if r.Method == http.MethodGet && len(reqpath) == 1 {
			http.ServeFileFS(w, r, dir.Fsys, reqpath[0])
			return false
		}

		http.NotFound(w, r)
		return false
	}
}

// GitReloadHandler returns a rate-limited handler which runs "git fetch" and "git reset --hard" in osDir, then reloads the server.
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
