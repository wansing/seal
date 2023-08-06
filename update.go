package seal

import (
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// not in UpdateHandler because UpdateHandler is called on each update
var updateHandlerLimiter = rate.NewLimiter(rate.Every(time.Minute), 2)

func (srv *Server) Update() {
	root, err := LoadDir(srv.Conf, nil, ".")
	if err != nil {
		log.Printf("error loading directory: %v", err)
	}
	srv.Root = root
}

func (srv *Server) UpdateHandler(filecontent []byte) Handler {
	secret := strings.TrimSpace(string(filecontent))
	return func(dir *Dir, reqpath []string, w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		if !updateHandlerLimiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("too many requests"))
			return
		}
		srv.Update()
		w.Write([]byte("ok"))
	}
}

// GitUpdateHandler returns a rate-limited handler which resets the local git repository to its upstream and then calls Update.
//
// We can't distinguish between local commits (which should be kept) and remote history rewerites (which can be dropped).
// Thus it accepts POST requests only and skips if there are local changes. You should not use it in interactive terminals and know "git reflog".
func (srv *Server) GitUpdateHandler(filecontent []byte) Handler {
	secret := strings.TrimSpace(string(filecontent))
	return func(dir *Dir, reqpath []string, w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("method not allowed"))
			return
		}
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		if !updateHandlerLimiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("too many requests"))
			return
		}

		localChanges, err := exec.Command("git", "status", "--porcelain").Output()
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
		if err := exec.Command("git", "fetch").Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error running git fetch"))
			return
		}
		if err := exec.Command("git", "reset", "--hard", "origin").Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error running git reset"))
			return
		}

		srv.Update()
		w.Write([]byte("ok"))
	}
}
