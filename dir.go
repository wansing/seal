package seal

import (
	"html/template"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

// A Dir represents a filesystem directory.
type Dir struct {
	Fsys    fs.FS  // allows for testing
	URLPath string // with leading slash
	// routing
	Subdirs map[string]*Dir
	// handling
	Handler  Handler
	Template *template.Template
}

// Load recursively loads dir.Subdirs, dir.Handler and dir.Template from dir.Fsys. If no handler is specified, the template handler is used.
func (dir *Dir) Load(config Config, parentTmpl *template.Template, urlpath string, errs *[]Error) error {
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

		ext := path.Ext(entry.Name())
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
			tmpl := templates.New(stem)
			err = contentFunc(urlpath, filecontent, tmpl)
			if err != nil {
				*errs = append(*errs, Error{urlpath, err.Error()})
			}
			continue
		}
	}

	// subdirs
	var subdirs = make(map[string]*Dir)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name() == "." || entry.Name() == ".." || strings.HasPrefix(entry.Name(), ".") { // skip hidden subdirs
			continue
		}

		subfsys, err := fs.Sub(dir.Fsys, entry.Name())
		if err != nil {
			return err
		}
		var subdir = &Dir{
			Fsys: subfsys,
		}

		urlitem := Slugify(entry.Name())
		if err := subdir.Load(config, templates, path.Join(urlpath, urlitem), errs); err != nil {
			return err
		}

		subdirs[entry.Name()] = subdir
	}

	dir.URLPath = urlpath
	dir.Subdirs = subdirs
	dir.Template = templates

	// generate handler at the end, when the rest of Dir is complete
	if handlerGen != nil {
		dir.Handler, err = handlerGen(dir, handlerGenFilestem, handlerGenFilecontent)
	} else {
		dir.Handler, err = MakeTemplateHandler(dir, "", nil)
	}
	if err != nil {
		*errs = append(*errs, Error{urlpath, err.Error()})
	}

	return nil
}

func Slugify(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = url.PathEscape(s) // just in case
	return s
}
