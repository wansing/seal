package seal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

var updateLimiter = rate.NewLimiter(rate.Every(time.Minute), 2)

type Server struct {
	Repo *Repository
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqpath := strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' })
	srv.Repo.Serve(reqpath, w, r)
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
		if err := srv.Repo.Update(nil); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(fmt.Sprintf("update took %d milliseconds", time.Since(start).Milliseconds())))
	}
}
