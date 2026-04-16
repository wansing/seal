package seal

import (
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
type ContentFunc func(t *template.Template, urlpath, fileroot string, filecontent []byte) error

type Error struct {
	URLPath string `json:"urlpath"`
	Err     error  `json:"error"`
}

// handler must handle full paths (including urlpath prefix)
type HandlerGen func(fsys fs.FS, urlpath string, t *template.Template, content map[string]ContentFunc) http.Handler

type Server struct {
	*http.ServeMux // not func (*Server) Handler() because we create a new handler on reload
	FS             fs.FS
	Content        map[string]ContentFunc // key is file extension
	Handlers       map[string]HandlerGen

	errs []Error
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

func (srv *Server) readDir(mux *http.ServeMux, tmpl *template.Template, fspath string, urlpath string) {
	entries, err := fs.ReadDir(srv.FS, fspath)
	if err != nil {
		srv.log(err, urlpath)
	}

	// read files
	var hasContent = false
	for _, entry := range entries {
		err := srv.readFile(mux, tmpl, fspath, urlpath, &hasContent, entry)
		if err != nil {
			srv.log(err, urlpath, entry.Name())
		}
	}

	// make "html" template default after it has been loaded, so that Execute works out of the box
	if h := tmpl.Lookup("html"); h != nil {
		tmpl = h
	}

	// use separate template for $
	dollarTmpl, _ := tmpl.Clone()

	// read files in $ subdir
	dollarEntries, _ := fs.ReadDir(srv.FS, path.Join(fspath, "$"))
	for _, entry := range dollarEntries {
		err := srv.readFile(mux, dollarTmpl, path.Join(fspath, "$"), urlpath, &hasContent, entry)
		if err != nil {
			srv.log(err, urlpath, entry.Name())
		}
	}

	// register template handler for this directory
	if hasContent {
		h, err := templateHandler(dollarTmpl, urlpath)
		if err != nil {
			srv.log(err, urlpath)
		}

		if urlpath == "/" {
			mux.HandleFunc("GET /{$}", h)
		} else {
			mux.HandleFunc("GET "+urlpath, h) // urlpath is without trailing slash, so it's not a prefix match
			mux.HandleFunc("GET "+urlpath+".html", redirectHTMLHandler)
		}
	}

	// subdirs
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "$" {
			continue
		}

		ext := path.Ext(entry.Name())
		switch {
		case ext == "":
			clonedTmpl, _ := tmpl.Clone() // always clone because we may have multiple subdirs
			srv.readDir(
				mux,
				clonedTmpl,
				path.Join(fspath, entry.Name()),
				path.Join(urlpath, MakeSlug(entry.Name())),
			)
		case srv.Handlers[ext] == nil:
			// skip unknown extension
		default:
			clonedTmpl, _ := tmpl.Clone() // always clone because we may have multiple subdirs
			subfs, _ := fs.Sub(srv.FS, entry.Name())
			suburlpath := path.Join(urlpath, strings.TrimSuffix(entry.Name(), ext))
			mux.Handle(suburlpath+"/", srv.Handlers[ext]( // trailing slash in order to to match subtree
				subfs,
				suburlpath,
				clonedTmpl,
				srv.Content,
			))
		}
	}
}

func (srv *Server) readFile(mux *http.ServeMux, tmpl *template.Template, fspath string, urlpath string, hasContent *bool, entry fs.DirEntry) error {
	if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
		return nil
	}

	// if extension is unknown, then serve as static file
	ext := path.Ext(entry.Name())
	if srv.Content[ext] == nil {
		mux.HandleFunc("GET "+path.Join(urlpath, entry.Name()), func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, srv.FS, path.Join(fspath, entry.Name()))
		})
		return nil
	}

	*hasContent = true
	fileroot := strings.TrimSuffix(entry.Name(), ext)
	filecontent, err := fs.ReadFile(srv.FS, path.Join(fspath, entry.Name()))
	if err != nil {
		return err
	}

	return srv.Content[ext](tmpl.New(fileroot), urlpath, fileroot, filecontent)
}

func (srv *Server) Reload() {
	srv.errs = srv.errs[:0]
	// don't mess with current mux (though live reload still won't be perfect, e. g. when deleting static files)
	mux := http.NewServeMux()
	srv.readDir(mux, template.New(""), ".", "/")
	srv.ServeMux = mux
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

// MakeSlug returns a modified version of the given string with [a-zA-Z0-9] retained and a dash in each gap.
func MakeSlug(s ...string) string {
	return fields(s, "-")
}

func MakeTemplateName(s ...string) string {
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
