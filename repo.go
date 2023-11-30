package seal

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/mattn/go-isatty"
)

type Repo struct {
	Conf Config
	Root *Dir
}

// Serve processes the given reqpath, calling the handler of each directory it passes by, until one handler returns false or the path is done.
//
// Serve always returns false. The return value exists for compatibility with type Handler.
func (repo *Repo) Serve(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
	dir := repo.Root
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

		// no subdir with that name found, but maybe a file
		if r.Method == http.MethodGet && dir.Files != nil {
			dir.Files.ServeHTTP(w, r)
			return false
		}

		http.NotFound(w, r)
		return false
	}
}

// Update reloads repo.Root using parentTmpl and repo.Conf.
func (repo *Repo) Update(parentTmpl *template.Template) error {
	dir := repo.Root
	conf := repo.Conf
	fspath := "."
	return dir.Load(conf, parentTmpl, fspath)
}

// MakeGitUpdateHandler returns a rate-limited handler which runs "git fetch" and "git reset --hard" in the repo directory, then updates the server.
//
// We can't distinguish between local commits (which should be kept) and upstream history rewrites (which can be dropped).
// Thus it fails if there are local changes and refuses to run from an interactive terminal.
// You should know about "git reflog".
func (repo *Repo) GitUpdateHandler(secret string, srv *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		if !updateLimiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("too many requests"))
			return
		}
		if repo.Root.FsPath == "" {
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte("no filesystem path given"))
			return
		}
		if isatty.IsTerminal(os.Stdout.Fd()) {
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte("git update has no effect when running in a terminal"))
			return
		}

		start := time.Now()
		status := exec.Command("git", "status", "--porcelain")
		status.Dir = repo.Root.FsPath
		localChanges, err := status.Output()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error running git status"))
			return
		}
		if len(localChanges) > 0 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("git working copy has local changes"))
			return
		}

		// https://stackoverflow.com/questions/9813816/git-pull-after-forced-update
		// this drops locals commits, however they can be restored with "git reflog" for a while
		fetch := exec.Command("git", "fetch")
		fetch.Dir = repo.Root.FsPath
		if err := fetch.Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error running git fetch"))
			return
		}
		reset := exec.Command("git", "reset", "--hard", "origin")
		reset.Dir = repo.Root.FsPath
		if err := reset.Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error running git reset"))
			return
		}

		if err := srv.Repository.Update(nil); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(fmt.Sprintf("git-update took %d milliseconds", time.Since(start).Milliseconds())))
	}
}
