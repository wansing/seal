package miniblog

import (
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/wansing/seal"
	"github.com/wansing/seal/handlers"
)

var isoDate = regexp.MustCompile("[0-9]{4}-[0-9]{2}-[0-9]{2}")

type postPreview struct {
	Anchor string
	Date   string
	Title  string
	URL    string
}

func Make(fsys fs.FS, urlpath string, t *template.Template, content map[string]seal.ContentFunc) http.Handler {
	indexTmpl, _ := t.Clone()
	indexTmpl.New("main").Parse(`
		<ul>
			{{range .}}
				<li id="{{.Anchor}}">
					<a href="{{.URL}}">{{.Date}} {{.Title}}</a>
				</li>
			{{end}}
		</ul>
	`)

	postTmpl, _ := t.Clone()
	postTmpl.New("main").Parse(`
		<p><a href="{{.BackURL}}">Back to Blog</a></p>
		<p>{{.Date}}</p>
		{{template "post" .}}
	`)

	var postFilenames []string
	entries, _ := fs.ReadDir(fsys, ".")
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		switch {
		case entry.IsDir():
			continue // skip dirs
		case strings.HasPrefix(entry.Name(), "."):
			continue // skip hidden files
		case len(entry.Name()) >= 10 && isoDate.MatchString(entry.Name()[:10]):
			postFilenames = append(postFilenames, entry.Name())
		case ext == ".html":
			// parse template and associate it to index page
			// not using ParseFS because we want the template name without filename extension
			filecontent, _ := fs.ReadFile(fsys, entry.Name())
			fileroot := strings.TrimSuffix(entry.Name(), ext)
			indexTmpl.New(fileroot).Parse(string(filecontent))
		}
	}
	slices.Sort(postFilenames)
	slices.Reverse(postFilenames)

	var mux = http.NewServeMux()
	var previews []postPreview

	mux.HandleFunc("GET "+urlpath+"/", func(w http.ResponseWriter, r *http.Request) {
		indexTmpl.Execute(w, previews)
	})

	for _, fn := range postFilenames {
		contentFunc, ok := content[path.Ext(fn)]
		if !ok {
			continue
		}
		filecontent, _ := fs.ReadFile(fsys, fn)
		fileroot := strings.TrimSuffix(fn, path.Ext(fn))
		date := fileroot[:10]

		tmpl, _ := postTmpl.Clone()
		_ = contentFunc(tmpl.New("post"), urlpath, fileroot, filecontent)

		// for blog index
		var title = handlers.Heading(tmpl.Lookup("post"))
		if title == "" {
			title = fileroot
		}
		previews = append(previews, postPreview{
			Anchor: fileroot,
			Date:   date,
			Title:  title,
			URL:    path.Join(urlpath, fileroot),
		})

		mux.HandleFunc("GET "+path.Join(urlpath, fileroot), func(w http.ResponseWriter, r *http.Request) {
			tmpl.Execute(w,
				struct {
					seal.TemplateData
					BackURL string
					Date    string
				}{
					TemplateData: seal.TemplateData{
						RequestURL: r.URL,
						URLPath:    path.Join(urlpath, fileroot),
					},
					BackURL: urlpath + "#" + seal.Slug(fileroot),
					Date:    date,
				},
			)
		})
	}

	return mux
}
