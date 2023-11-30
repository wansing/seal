package seal

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var errExecuteTemplate = template.Must(template.New("").Parse(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error executing template: {{.}}</p>`))

var errParsingTemplate = template.Must(template.New("").Parse(`<p style="border: solid red 2px; border-radius: 8px; padding: 12px">Error parsing template: {{.}}</p>`))

// execErrParsingTemplate safely wraps an error into an html string
func execErrParsingTemplate(err error) string {
	var buf bytes.Buffer
	errParsingTemplate.Execute(&buf, err)
	return buf.String()
}

// A Dir represents a filesystem directory.
type Dir struct {
	Fsys   fs.FS  // use fs for testability
	FsPath string // preserve path for git-update
	// routing
	Subdirs map[string]*Dir
	// handling
	Files    http.Handler
	Handler  Handler
	Template *template.Template
}

func MakeDir(fspath string) *Dir {
	return &Dir{
		Fsys:   os.DirFS(fspath),
		FsPath: fspath,
	}
}

// Load recursively loads dir.Subdirs, dir.Files, dir.Handler and dir.Template from dir.Fsys. If no handler is specified, the template handler is used.
func (dir *Dir) Load(config Config, parentTmpl *template.Template, fspath string) error {
	if parentTmpl == nil {
		parentTmpl = template.New("")
	}

	entries, err := fs.ReadDir(dir.Fsys, ".")
	if err != nil {
		return err
	}

	// files
	var handlerGen HandlerGen
	var handlerGenFilestem string    // HandlerGen argument
	var handlerGenFilecontent []byte // HandlerGen argument
	var templates, _ = parentTmpl.Clone()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		stem := strings.TrimSuffix(entry.Name(), ext)

		// Handlers[filename]
		if gen, ok := config.Handlers[entry.Name()]; ok {
			filecontent, err := fs.ReadFile(dir.Fsys, entry.Name())
			if err != nil {
				return err
			}
			handlerGen = gen
			handlerGenFilecontent = filecontent
			continue
		}

		// Handlers[ext]
		if gen, ok := config.Handlers[ext]; ok {
			filecontent, err := fs.ReadFile(dir.Fsys, entry.Name())
			if err != nil {
				return err
			}
			handlerGen = gen
			handlerGenFilestem = stem
			handlerGenFilecontent = filecontent
			continue
		}

		// Content
		if contentFunc, ok := config.Content[ext]; ok {
			filecontent, err := fs.ReadFile(dir.Fsys, entry.Name())
			if err != nil {
				return err
			}

			dirpath := strings.TrimSuffix(path.Join("/", fspath), "/") // root becomes "", so the html code can append "/" without getting "//"
			tmpl := templates.New(stem)

			err = contentFunc(dirpath, filecontent, tmpl)
			if err != nil {
				tmpl.Parse(execErrParsingTemplate(err))
			}
			continue
		}
	}

	// subdirs
	var subdirs = make(map[string]*Dir)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		subfsys, err := fs.Sub(dir.Fsys, entry.Name())
		if err != nil {
			return err
		}
		var subdir = &Dir{
			Fsys: subfsys,
		}
		if err := subdir.Load(config, templates, filepath.Join(fspath, entry.Name())); err != nil {
			return err
		}

		subdirs[entry.Name()] = subdir
	}

	dir.Files = http.FileServer(http.FS(dir.Fsys)) // same for each Dir, better use ServeFileFS when it's in the standard library
	dir.Subdirs = subdirs
	dir.Template = templates

	// generate handler at the end, when the rest of Dir is complete
	if handlerGen != nil {
		dir.Handler = handlerGen(dir, handlerGenFilestem, handlerGenFilecontent)
	} else {
		dir.Handler = MakeTemplateHandler(dir, "", nil)
	}

	return nil
}
