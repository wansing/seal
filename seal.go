package seal

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type Ext func(filecontent []byte) string

type HandlerGen func(filecontent []byte) Handler

type Handler func(dir *Dir, reqpath *[]string, w http.ResponseWriter, r *http.Request)

func handleTemplate(dir *Dir, _ *[]string, w http.ResponseWriter, r *http.Request) {
	err := dir.Template.ExecuteTemplate(w, "html", nil)
	if err != nil {
		log.Println(err)
	}
}

// Seal is both the configuration and the http handler. This is because Filenames["update"] modifies the DirHandler.
type Seal struct {
	Fsys      fs.FS
	Exts      map[string]Ext        // key: file extension, e.g. ".md"
	Filenames map[string]HandlerGen // key: file name, e.g. "redirect"
	Params    map[string]Handler    // key: directory name, e.g. "{date}"

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

		// Exts

		ext := filepath.Ext(entry.Name())
		fn, ok := s.Exts[ext]
		if !ok {
			continue
		}

		templateHandler = true

		filecontent, err := fs.ReadFile(s.Fsys, entrypath)
		if err != nil {
			return nil, err
		}
		htm := fn(filecontent)

		// We want to create links and embed images which are relative to a directory.
		//
		// Rewriting <a href="..."> and <img src="..."> is hard because:
		// Execution puts templates from different directories together.
		// So we can only manipulate them before execution, leaving two options:
		// Parsing as HTML would be hard because the renderer would escape quotes in template actions.
		// Modifying the template parse tree would also be hard because it does not parse HTML.
		//
		// Instead, let's pass the directory to the template through a variable $dir.
		// We have to inject the template action before parsing, else parsing fails when trying to access $dir.
		// This does only work for the main template, not for {{define}}'d templates.
		dirpath := strings.TrimSuffix(path.Join("/", fspath), "/") // root becomes "", so the html code can append "/" without getting "//"
		htm = `{{$dir := "` + dirpath + `"}}` + "\n" + htm

		tmplName := strings.TrimSuffix(entry.Name(), ext)
		ts, err := template.New(tmplName).Parse(htm)
		if err != nil {
			ts, _ = template.New(tmplName).Parse(fmt.Sprintf(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error parsing template: %v</p>`, err))
		}

		for _, t := range ts.Templates() {
			if t.Tree != nil {
				_, err := templates.AddParseTree(t.Name(), t.Tree) // returned template has t.Name() and t.Tree, does not matter because the template set is the same
				if err != nil {
					log.Println(err)
				}
			}
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
