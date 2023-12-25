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
	Errs []Error
	Repo *Repository
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

func (srv *Server) ErrorsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(srv.Errs)
	}
}

func (srv *Server) Update() error {
	var errs = []Error{} // initialize it to get json "[]" instead of "null"
	if err := srv.Repo.Update(nil, &errs); err != nil {
		return err
	}
	srv.Errs = errs
	return nil
}

func (srv *Server) UpdateHandler(secret string) http.HandlerFunc {
	limitedUpdate := Limit(time.Minute, 2, func() {
		srv.Update() // ignore returned error
	})
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}

		start := time.Now()
		if done := limitedUpdate(); done {
			w.Write([]byte(fmt.Sprintf("update took %d milliseconds", time.Since(start).Milliseconds())))
		} else {
			w.Write([]byte("update scheduled"))
		}
	}
}
