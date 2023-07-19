package seal

import (
	"bytes"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type TemplateGen func(dirpath string, filecontent []byte, tmpl *template.Template) error

type HandlerGen func(filecontent []byte) Handler

type Handler func(dir *Dir, reqpath *[]string, w http.ResponseWriter, r *http.Request)

var errExecuteTemplate = template.Must(template.New("").Parse(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error executing template: {{.}}</p>`))

var errParsingTemplate = template.Must(template.New("").Parse(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error parsing template: {{.}}</p>`))

// execErrParsingTemplate safely wraps the error into an html string
func execErrParsingTemplate(err error) string {
	var buf bytes.Buffer
	errParsingTemplate.Execute(&buf, err)
	return buf.String()
}

func handleTemplate(dir *Dir, _ *[]string, w http.ResponseWriter, r *http.Request) {
	err := dir.Template.ExecuteTemplate(w, "html", nil)
	if err != nil {
		errExecuteTemplate.Execute(w, err)
	}
}

// Seal is both the configuration and the http handler. This is because Filenames["update"] modifies the DirHandler.
type Seal struct {
	Fsys      fs.FS
	FileExts  map[string]TemplateGen
	Filenames map[string]HandlerGen
	Params    map[string]Handler // key: directory name, e.g. "{date}"

	RootHandler DirHandler
}

func (s *Seal) ListenAndServe(addr string) {
	s.RootHandler = DirHandler{
		Files: http.FileServer(http.FS(s.Fsys)), // better use ServeFileFS when it's in the standard library
	}
	s.Update()
	log.Printf("listening to %s", addr)
	http.ListenAndServe(addr, &s.RootHandler)
}

// LoadDir recursively loads the given filesystem into a *Dir.
func (s *Seal) LoadDir(parentTmpl *template.Template, fspath string) (*Dir, error) {
	if parentTmpl == nil {
		parentTmpl = template.New("")
	}

	entries, err := fs.ReadDir(s.Fsys, fspath)
	if err != nil {
		return nil, err
	}

	// files first, build templates
	var templates, _ = parentTmpl.Clone()
	var filenameHandler Handler
	var templateHandler bool
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entrypath := filepath.Join(fspath, entry.Name())

		// Filenames

		if gen, ok := s.Filenames[entry.Name()]; ok {
			filecontent, err := fs.ReadFile(s.Fsys, entrypath)
			if err != nil {
				return nil, err
			}
			filenameHandler = gen(filecontent)
			continue
		}

		// FileExts

		ext := filepath.Ext(entry.Name())
		fn, ok := s.FileExts[ext]
		if !ok {
			continue
		}

		templateHandler = true

		filecontent, err := fs.ReadFile(s.Fsys, entrypath)
		if err != nil {
			return nil, err
		}

		dirpath := strings.TrimSuffix(path.Join("/", fspath), "/") // root becomes "", so the html code can append "/" without getting "//"
		tmplName := strings.TrimSuffix(entry.Name(), ext)
		tmpl := templates.New(tmplName)

		err = fn(dirpath, filecontent, tmpl)
		if err != nil {
			tmpl.Parse(execErrParsingTemplate(err))
		}
	}

	// subdirs
	var defaultSubdir string
	var subdirs = make(map[string]*Dir)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		entrypath := filepath.Join(fspath, entry.Name())
		subdir, err := s.LoadDir(templates, entrypath)
		if err != nil {
			return nil, err
		}

		if middlewareHandler, ok := s.Params[entry.Name()]; ok {
			subdir.MiddlewareHandler = middlewareHandler
			defaultSubdir = entry.Name()
		}

		subdirs[entry.Name()] = subdir
	}

	var handler Handler
	if templateHandler {
		handler = handleTemplate
	}
	if filenameHandler != nil {
		handler = filenameHandler // overwrite template handler
	}

	return &Dir{
		Subdirs:       subdirs,
		DefaultSubdir: defaultSubdir,
		Handler:       handler,
		Template:      templates,
	}, nil
}

func (s *Seal) Update() {
	root, err := s.LoadDir(nil, ".")
	if err != nil {
		log.Printf("error loading directory: %v", err)
	}
	s.RootHandler.Root = root
}

func (s *Seal) UpdateHandler(filecontent []byte) Handler {
	// most webhooks are POST
	secret := strings.TrimSpace(string(filecontent))
	return func(dir *Dir, reqpath *[]string, w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("secret") == secret {
			s.Update()
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

type DirHandler struct {
	Root  *Dir
	Files http.Handler
}

// ServeHTTP implements http.Handler.
func (h *DirHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = path.Clean(r.URL.Path)

	var reqpath = strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' })
	var dir = h.Root
	for it := 0; len(reqpath) > 0 && it < 16; it++ {
		next, ok := dir.Subdirs[reqpath[0]]
		if ok {
			reqpath = reqpath[1:]
		} else {
			next, ok = dir.Subdirs[dir.DefaultSubdir]
			if !ok {
				if r.Method == http.MethodGet && h.Files != nil {
					h.Files.ServeHTTP(w, r)
					return
				} else {
					http.NotFound(w, r)
					return
				}
			}
		}

		dir = next

		// may modify reqpath, so we run it before the for condition is checked
		if dir.MiddlewareHandler != nil {
			dir.MiddlewareHandler(dir, &reqpath, w, r)
		}
	}

	if dir.Handler != nil {
		dir.Handler(dir, &reqpath, w, r)
	} else {
		http.NotFound(w, r)
	}
}

// A Dir is generated from a filesystem directory. It has no knowledge about request-scoped {parameter} values.
type Dir struct {
	// routing
	Subdirs           map[string]*Dir
	DefaultSubdir     string
	MiddlewareHandler Handler
	// handling
	Handler  Handler
	Template *template.Template // Contains templates from parent directories. Don't hide in Handler, may need it later.
}
