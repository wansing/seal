package seal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

var updateLimiter = rate.NewLimiter(rate.Every(time.Minute), 2)

type Error struct {
	URLPath string `json:"urlpath"`
	Message string `json:"message"`
}

type Server struct {
	Errs []Error
	Repo *Repository
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		start := time.Now()
		if err := srv.Update(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(fmt.Sprintf("update took %d milliseconds", time.Since(start).Milliseconds())))
	}
}
