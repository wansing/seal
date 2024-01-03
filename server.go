package seal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Error struct {
	URLPath string `json:"urlpath"`
	Message string `json:"message"`
}

type Server struct {
	Repo *Repository
	Errs []Error
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// remove trailing slash in GET requests, except for root
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
		http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/"), http.StatusMovedPermanently)
		return
	}

	reqpath := strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' })
	srv.Repo.Serve(reqpath, w, r)
}

// ErrorsHandler returns a handler which sends srv.Errs in JSON.
func (srv *Server) ErrorsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(srv.Errs)
	}
}

// Reload calls srv.Repo.Reload and updates srv.Errs.
func (srv *Server) Reload() error {
	var errs = []Error{} // initialize it to get json "[]" instead of "null"
	if err := srv.Repo.Reload(nil, &errs); err != nil {
		return err
	}
	srv.Errs = errs
	return nil
}

// ReloadHandler returns a rate-limited handler which calls srv.Reload.
func (srv *Server) ReloadHandler(secret string) http.HandlerFunc {
	limitedReload := Limit(time.Minute, 2, func() {
		srv.Reload() // ignore returned error
	})
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}

		start := time.Now()
		if done := limitedReload(); done {
			w.Write([]byte(fmt.Sprintf("reload took %d milliseconds", time.Since(start).Milliseconds())))
		} else {
			w.Write([]byte("reload scheduled"))
		}
	}
}
