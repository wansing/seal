package seal

import (
	"html/template"
	"io/fs"
	"path"
	"strings"
)

// A Dir represents a filesystem directory.
type Dir struct {
	Data    map[string]string
	Fsys    fs.FS  // allows for testing
	URLPath string // with leading slash
	// routing
	Subdirs map[string]*Dir
	// handling
	Handler  Handler
	Template *template.Template
}

// HasTemplate returns whether dir.Templates contains a non-empty subtemplate with the given name.
//
// Note that templates are first parsed and then excecuted, and that HasTemplate is called during execution.
// Parsing will fail if a subtemplate is not defined, even if it is not executed later.
// You can use {{block}} to define an empty fallback template:
//
//	{{if .HasTemplate "preface"}}
//	  <h1>Preface</h1>
//	  {{block "preface" .}}{{end}}
//	{{end}}
func (dir *Dir) HasTemplate(name string) bool {
	t := dir.Template.Lookup(name)
	return t != nil && t.Tree != nil && t.Tree.Root != nil && len(t.Tree.Root.Nodes) > 0
}

// String returns dir.URLPath with a trailing slash.
func (dir *Dir) String() string {
	return path.Join(dir.URLPath, "/")
}

// Load creates a *Dir from the given fsys.
func Load(config Config, parentTmpl *template.Template, fsys fs.FS, urlpath string, errs *[]Error) (*Dir, error) {
	if parentTmpl == nil {
		parentTmpl = template.New("")
	}

	tmpl, _ := parentTmpl.Clone()
	dir := &Dir{
		Data:     make(map[string]string),
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
		subdir, err := Load(config, dir.Template, subfsys, path.Join(urlpath, Slugify(entry.Name())), errs)
		if err != nil {
			return nil, err
		}
		dir.Subdirs[entry.Name()] = subdir
	}

	// make dir.Template.Execute work without specifying a template name
	if h := dir.Template.Lookup("html"); h != nil {
		dir.Template = h
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

// Slugify returns a modified version of the given string with [a-zA-Z0-9] retained and a dash in each gap.
func Slugify(s string) string {
	strs := strings.FieldsFunc(s, func(r rune) bool {
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
	})
	return strings.Join(strs, "-")
}
