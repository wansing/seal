package seal

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type Error struct {
	URLPath string `json:"urlpath"`
	Message string `json:"message"`
}

type Server struct {
	Content  map[string]ContentFunc // key is file extension
	Handlers map[string]HandlerGen  // key is file extension or full filename
	FS       *FS
	Errs     []Error
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// remove trailing slash in GET requests, except for root
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
		http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/"), http.StatusMovedPermanently)
		return
	}

	reqpath := strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' })
	srv.FS.Serve(reqpath, w, r)
}

// ErrorsHandler returns a handler which sends srv.Errs in JSON.
func (srv *Server) ErrorsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(srv.Errs)
	}
}

func (srv *Server) Reload() error {
	var errs = []Error{} // initialize it to get json "[]" instead of "null"
	defer func() {
		srv.Errs = errs
	}()
	return srv.ReloadFS(srv.FS, nil, &errs)
}

// Reload updates fs.Root.
func (srv *Server) ReloadFS(fs *FS, parent *Dir, errs *[]Error) error {
	var parentTmpl *template.Template
	var baseURLPath = "/"
	if parent != nil {
		parentTmpl = parent.Template
		baseURLPath = parent.URLPath
	}

	dir, err := srv.Load(parentTmpl, fs.Fsys, baseURLPath, errs)
	if err != nil {
		return err
	}
	fs.root = dir
	return nil
}

// ReloadHandler returns a rate-limited handler which calls srv.Reload.
func (srv *Server) ReloadHandler(secret string) http.HandlerFunc {
	limitedReload := Limit(time.Minute, 2, func() error {
		return srv.Reload()
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
