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

// Load creates a *Dir from the given fsys.
func Load(config Config, parentTmpl *template.Template, fsys fs.FS, urlpath string, errs *[]Error) (*Dir, error) {
	if parentTmpl == nil {
		parentTmpl = template.New("")
	}

	tmpl, _ := parentTmpl.Clone()
	dir := &Dir{
		Fsys:     fsys,
		URLPath:  urlpath,
		Template: tmpl,
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	// files
	var containsContent = false
	var handlerGen HandlerGen
	var handlerGenFilestem string    // HandlerGen argument
	var handlerGenFilecontent []byte // HandlerGen argument
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := path.Ext(entry.Name())
		stem := strings.TrimSuffix(entry.Name(), ext)

		// Handlers[filename]
		if gen, ok := config.Handlers[entry.Name()]; ok {
			filecontent, err := fs.ReadFile(fsys, entry.Name())
			if err != nil {
				return nil, err
			}
			handlerGen = gen
			handlerGenFilecontent = filecontent
			continue
		}

		// Handlers[ext]
		if gen, ok := config.Handlers[ext]; ok {
			filecontent, err := fs.ReadFile(fsys, entry.Name())
			if err != nil {
				return nil, err
			}
			handlerGen = gen
			handlerGenFilestem = stem
			handlerGenFilecontent = filecontent
			continue
		}

		// Content
		if contentFunc, ok := config.Content[ext]; ok {
			containsContent = true
			filecontent, err := fs.ReadFile(fsys, entry.Name())
			if err != nil {
				return nil, err
			}
			err = contentFunc(dir, stem, filecontent)
			if err != nil {
				*errs = append(*errs, Error{urlpath, err.Error()})
			}
			continue
		}
	}

	// subdirs
	dir.Subdirs = make(map[string]*Dir)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name() == "." || entry.Name() == ".." || strings.HasPrefix(entry.Name(), ".") { // skip hidden subdirs
			continue
		}

		subfsys, err := fs.Sub(fsys, entry.Name())
		if err != nil {
			return nil, err
		}
		subdir, err := Load(config, tmpl, subfsys, path.Join(urlpath, Slugify(entry.Name())), errs)
		if err != nil {
			return nil, err
		}
		dir.Subdirs[entry.Name()] = subdir
	}

	// generate handler at the end, when the rest of Dir is complete
	if handlerGen != nil {
		dir.Handler, err = handlerGen(dir, handlerGenFilestem, handlerGenFilecontent)
	} else {
		if containsContent {
			dir.Handler, err = MakeTemplateHandler(dir, "", nil)
		}
		// else no template handler because it would probably display duplicate content
	}
	if err != nil {
		*errs = append(*errs, Error{urlpath, err.Error()})
	}

	return dir, nil
}

func Slugify(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = url.PathEscape(s) // just in case
	return s
}
