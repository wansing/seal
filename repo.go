package seal

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/mattn/go-isatty"
)

type Repository struct {
	Conf    Config
	Fsys    fs.FS
	RootDir string // for git reload

	root *Dir // set by Reload
}

func MakeDirRepository(config Config, dir string) *Repository {
	return &Repository{
		Conf:    config,
		Fsys:    os.DirFS(dir),
		RootDir: dir,
	}
}

// Serve processes the given reqpath, calling the handler of each directory it passes by, until one handler returns false or the path is done.
//
// Serve always returns false. The return value exists for compatibility with type Handler.
func (repo *Repository) Serve(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
	dir := repo.root
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
		if r.Method == http.MethodGet && len(reqpath) == 1 {
			http.FileServer(http.FS(repo.Fsys)).ServeHTTP(w, r) // only works with server repository at the moment
			// http.ServeFileFS(w, r, dir.Fsys, reqpath[0]) // use this from go 1.22
			return false
		}

		http.NotFound(w, r)
		return false
	}
}

// Reload updates repo.Root.
func (repo *Repository) Reload(parent *Dir, errs *[]Error) error {
	var parentTmpl *template.Template
	var baseURLPath = "/"
	if parent != nil {
		parentTmpl = parent.Template
		baseURLPath = parent.URLPath
	}

	dir, err := Load(repo.Conf, parentTmpl, repo.Fsys, baseURLPath, errs)
	if err != nil {
		return err
	}
	repo.root = dir
	return nil
}

// GitReloadHandler returns a rate-limited handler which runs "git fetch" and "git reset --hard" in the repo root dir, then reloads the server.
//
// We can't distinguish between local commits (which should be kept) and upstream history rewrites (which can be dropped).
// Thus it fails if there are local changes and refuses to run from an interactive terminal.
// You should know about "git reflog".
func (repo *Repository) GitReloadHandler(secret string, srv *Server) http.HandlerFunc {
	if repo.RootDir == "" {
		return http.NotFound
	}
	limitedReload := Limit(time.Minute, 2, func() {
		srv.Reload() // ignore returned error
	})
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		if isatty.IsTerminal(os.Stdout.Fd()) {
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte("git reload has no effect when running in a terminal"))
			return
		}

		start := time.Now()
		status := exec.Command("git", "status", "--porcelain")
		status.Dir = repo.RootDir
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
		fetch.Dir = repo.RootDir
		if err := fetch.Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error running git fetch: %v", err)
			return
		}
		reset := exec.Command("git", "reset", "--hard", "origin")
		reset.Dir = repo.RootDir
		if err := reset.Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error running git reset: %v", err)
			return
		}

		if done := limitedReload(); done {
			w.Write([]byte(fmt.Sprintf("git reload took %d milliseconds", time.Since(start).Milliseconds())))
		} else {
			w.Write([]byte("git reload scheduled"))
		}
	}
}
