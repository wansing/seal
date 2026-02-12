package seal

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// A ContentFunc populates the template t.
// The urlpath can be used to make relative links absolute.
// The fileroot is useful to distinguish between multiple instances of this content on the same page.
type ContentFunc func(t *template.Template, urlpath, fileroot string, filecontent []byte, broker *Broker) error

type Error struct {
	URLPath string `json:"urlpath"`
	Err     error  `json:"error"`
}

// handler must handle full paths (including urlpath prefix)
type HandlerGen func(fsys fs.FS, urlpath string, t *template.Template, content map[string]ContentFunc, broker *Broker) http.Handler

type Server struct {
	*http.ServeMux // not func (*Server) Handler() because we create a new handler on reload
	FS             fs.FS
	Content        map[string]ContentFunc // key is file extension
	Handlers       map[string]HandlerGen

	broker *Broker
	errs   []Error
	files  map[string]string // urlpath => fspath
}

func (srv *Server) log(err error, urlpath ...string) {
	log.Printf("%s: %v", path.Join(urlpath...), err)
	srv.errs = append(srv.errs, Error{
		URLPath: path.Join(urlpath...),
		Err:     err,
	})
}

// ErrorsHandler returns a handler which sends srv.errs in JSON.
func (srv *Server) ErrorsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errs = srv.errs
		if errs == nil {
			errs = []Error{} // json "[]" instead of "null"
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(errs)
	}
}

func (srv *Server) LoadDir(tmpl *template.Template, fspath string, urlpath string) {
	entries, err := fs.ReadDir(srv.FS, fspath)
	if err != nil {
		srv.log(err, urlpath)
	}

	// files first
	var hasContent = false
	for _, entry := range entries {
		ext := path.Ext(entry.Name())
		switch {
		case entry.IsDir():
			continue // later
		case strings.HasPrefix(entry.Name(), "."):
			continue // skip hidden files
		case srv.Content[ext] == nil:
			srv.files[path.Join(urlpath, entry.Name())] = path.Join(fspath, entry.Name())
		default:
			filecontent, err := fs.ReadFile(srv.FS, path.Join(fspath, entry.Name()))
			if err != nil {
				srv.log(err, urlpath, entry.Name())
			}
			// Template parse functions ignore template definitions "with a body containing only white space and comments", so we require non-whitespaces at least.
			if len(bytes.TrimSpace(filecontent)) > 0 {
				hasContent = true
			}
			fileroot := strings.TrimSuffix(entry.Name(), ext)
			err = srv.Content[ext](tmpl.New(fileroot), urlpath, fileroot, filecontent, srv.broker) // leaks fileroot
			if err != nil {
				srv.log(err, urlpath, entry.Name())
			}
		}
	}

	// make tmpl.Execute work without specifying a template name
	if h := tmpl.Lookup("html"); h != nil {
		tmpl = h
	}

	// register template handler for this directory
	if hasContent {
		h, err := templateHandler(tmpl, urlpath)
		if err != nil {
			srv.log(err, urlpath)
		}

		if urlpath == "/" {
			srv.ServeMux.HandleFunc("GET /{$}", h)
		} else {
			srv.ServeMux.HandleFunc("GET "+urlpath, h) // urlpath is without trailing slash
			srv.ServeMux.HandleFunc("GET "+urlpath+".html", redirectHTMLHandler)
		}
	}

	// subdirs
	for _, entry := range entries {
		ext := path.Ext(entry.Name())
		switch {
		case !entry.IsDir():
			continue // files already done
		case strings.HasPrefix(entry.Name(), "."):
			continue // skip hidden dirs
		case ext == "":
			clonedTmpl, _ := tmpl.Clone()
			srv.LoadDir(
				clonedTmpl,
				path.Join(fspath, entry.Name()),
				path.Join(urlpath, Slug(entry.Name())),
			)
		case srv.Handlers[ext] == nil:
			// skip unknown extension
		default:
			subfs, _ := fs.Sub(srv.FS, entry.Name())
			suburlpath := path.Join(urlpath, strings.TrimSuffix(entry.Name(), ext))
			srv.ServeMux.Handle(suburlpath+"/", srv.Handlers[ext]( // trailing slash in order to to match subtree
				subfs,
				suburlpath,
				tmpl,
				srv.Content,
				srv.broker,
			))
		}
	}
}

func (srv *Server) Reload() {
	srv.broker = NewBroker()
	srv.errs = srv.errs[:0]
	srv.files = make(map[string]string)
	srv.ServeMux = http.NewServeMux()
	srv.LoadDir(template.New(""), ".", "/")
	srv.ServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if fspath, ok := srv.files[r.URL.Path]; ok {
			http.ServeFileFS(w, r, srv.FS, fspath)
		} else {
			http.NotFound(w, r)
		}
	})
	srv.broker.Ready()
}

type TemplateData struct {
	RequestURL *url.URL // not the full request because that may leak cookies
	URLPath    string
}

// internalServerError replies to the request with an HTTP 500 internal server error.
func internalServerError(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "500 internal server error", http.StatusInternalServerError)
}

func redirectHTMLHandler(w http.ResponseWriter, r *http.Request) {
	if p, ok := strings.CutSuffix(r.URL.Path, ".html"); ok {
		http.Redirect(w, r, p, http.StatusSeeOther)
	} else {
		http.NotFound(w, r)
	}
}

func templateHandler(tmpl *template.Template, urlpath string) (http.HandlerFunc, error) {
	// test template execution, clone before so template can be extended later
	t, err := tmpl.Clone()
	if err != nil {
		return internalServerError, err
	}
	if err := t.Execute(io.Discard, TemplateData{
		RequestURL: &url.URL{Path: urlpath},
		URLPath:    urlpath,
	}); err != nil {
		return internalServerError, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, TemplateData{
			RequestURL: r.URL,
			URLPath:    urlpath,
		}) // ignore error, assume that initial execution test was enough
	}, nil
}

// Slug returns a modified version of the given string with [a-zA-Z0-9] retained and a dash in each gap.
func Slug(s ...string) string {
	return fields(s, "-")
}

func TemplateName(s ...string) string {
	return fields(s, "_")
}

func fields(strs []string, sep string) string {
	var fields []string
	for _, s := range strs {
		fields = append(fields, strings.FieldsFunc(s, func(r rune) bool {
			if 'a' <= r && r <= 'z' {
				return false
			}
			if 'A' <= r && r <= 'Z' {
				return false
			}
			if '0' <= r && r <= '9' {
				return false
			}
			return true
		})...)
	}
	return strings.Join(fields, sep)
}
