package seal

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

type Error struct {
	URLPath string `json:"urlpath"`
	Message string `json:"message"`
}

type Server struct {
	FS       fs.FS
	Content  map[string]ContentFunc // key is file extension
	Handlers map[string]HandlerGen  // key is file extension or full filename

	root *Dir
	errs []Error
}

// ServeHTTP processes the request path, calling the handler of each directory it passes by, until one handler returns false or the path is done.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// remove trailing slash in GET requests, except for root
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
		http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/"), http.StatusMovedPermanently)
		return
	}

	dir := srv.root
	reqpath := strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' })
	for {
		if dir.Handler != nil {
			cont := dir.Handler(reqpath, w, r)
			if !cont {
				return
			}
		}

		if len(reqpath) == 0 {
			http.NotFound(w, r)
			return
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
			return
		}

		http.NotFound(w, r)
		return
	}
}

// ErrorsHandler returns a handler which sends srv.errs in JSON.
func (srv *Server) ErrorsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if srv.errs == nil {
			srv.errs = []Error{} // initialize it to get json "[]" instead of "null"
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(srv.errs)
	}
}

func (srv *Server) Reload() {
	srv.root, srv.errs = srv.Load(nil, srv.FS, "/")
}
