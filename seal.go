package seal

import (
	"bytes"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type TemplateGen func(dirpath string, filecontent []byte, tmpl *template.Template) error

type HandlerGen func(dir *Dir, filecontent []byte) Handler

// Handler returns true in order to continue execution, false to stop execution.
type Handler func(reqpath []string, w http.ResponseWriter, r *http.Request) bool

var errExecuteTemplate = template.Must(template.New("").Parse(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error executing template: {{.}}</p>`))

var errParsingTemplate = template.Must(template.New("").Parse(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error parsing template: {{.}}</p>`))

// execErrParsingTemplate safely wraps the error into an html string
func execErrParsingTemplate(err error) string {
	var buf bytes.Buffer
	errParsingTemplate.Execute(&buf, err)
	return buf.String()
}

// Template returns a Handler which executes dir.Template if the request path has been consumed.
func Template(dir *Dir, _ []byte) Handler {
	return func(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
		if len(reqpath) > 0 {
			return true
		}

		if dir.TemplateDiffers {
			dir.ExecuteTemplate(w, "html")
		} else {
			http.NotFound(w, r) // would be duplicate content
		}
		return false
	}
}

// A Dir is generated from a filesystem directory. It has no knowledge about request-scoped {parameter} values.
type Dir struct {
	// routing
	Subdirs map[string]*Dir
	// handling
	Files           http.Handler       // copy of Server.Files
	Handler         Handler            // never nil
	Template        *template.Template // never nil
	TemplateDiffers bool               // differs from parent template
}

// LoadDir recursively loads the given filesystem into a *Dir.
func LoadDir(config Config, parentTmpl *template.Template, fspath string) (*Dir, error) {
	if parentTmpl == nil {
		parentTmpl = template.New("")
	}

	entries, err := fs.ReadDir(config.Fsys, fspath)
	if err != nil {
		return nil, err
	}

	// files
	var handlerGen HandlerGen
	var handlerGenFilecontent []byte
	var templates, _ = parentTmpl.Clone()
	var templateDiffers = false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entrypath := filepath.Join(fspath, entry.Name())

		// Filenames

		if gen, ok := config.Filenames[entry.Name()]; ok {
			filecontent, err := fs.ReadFile(config.Fsys, entrypath)
			if err != nil {
				return nil, err
			}
			handlerGen = gen // don't call gen() here because Dir is not complete yet
			handlerGenFilecontent = filecontent
			continue
		}

		// FileExts

		ext := filepath.Ext(entry.Name())
		fn, ok := config.FileExts[ext]
		if !ok {
			continue
		}

		templateDiffers = true

		filecontent, err := fs.ReadFile(config.Fsys, entrypath)
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
	var subdirs = make(map[string]*Dir)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		entrypath := filepath.Join(fspath, entry.Name())
		subdir, err := LoadDir(config, templates, entrypath)
		if err != nil {
			return nil, err
		}

		subdirs[entry.Name()] = subdir
	}

	// without Dir.Handler
	dir := &Dir{
		Files:           http.FileServer(http.FS(config.Fsys)), // same for each Dir, better use ServeFileFS when it's in the standard library
		Subdirs:         subdirs,
		Template:        templates,
		TemplateDiffers: templateDiffers,
	}

	if handlerGen != nil {
		dir.Handler = handlerGen(dir, handlerGenFilecontent)
	} else {
		dir.Handler = Template(dir, nil)
	}

	return dir, nil
}

// for embedding content (e.g. blog post preview) without executing their other (redirect etc.) handlers
func (dir *Dir) ExecuteTemplate(w io.Writer, name string) {
	err := dir.Template.ExecuteTemplate(w, name, dir)
	if err != nil {
		errExecuteTemplate.Execute(w, err)
	}
}

type Config struct {
	Fsys      fs.FS
	FileExts  map[string]TemplateGen
	Filenames map[string]HandlerGen
}

type Server struct {
	Conf Config
	Root *Dir
}

func (srv *Server) ListenAndServe(addr string) {
	srv.Update()
	log.Printf("listening to %s", addr)
	http.ListenAndServe(addr, srv)
}

// ServeHTTP implements http.Handler.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = path.Clean(r.URL.Path)
	reqpath := strings.FieldsFunc(r.URL.Path, func(r rune) bool { return r == '/' })

	dir := srv.Root
	for {
		cont := dir.Handler(reqpath, w, r)
		if !cont {
			return
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

		// no subdir with that name found, but maybe a file
		if r.Method == http.MethodGet && dir.Files != nil {
			dir.Files.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
		return
	}
}
