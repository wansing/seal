package miniblog

import (
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/wansing/seal"
	"github.com/wansing/seal/content"
	"github.com/wansing/seal/handlers"
)

var isoDate = regexp.MustCompile("[0-9]{4}-[0-9]{2}-[0-9]{2}")

type Miniblog struct {
	previews []postPreview
}

type postPreview struct {
	Anchor string
	Date   string
	Title  string
	URL    string
}

type IndexData struct {
	seal.TemplateData
	Previews []postPreview
}

type PostData struct {
	seal.TemplateData
	BackURL string
	Date    string
}

func (mb *Miniblog) Latest(t *template.Template, urlpath, fileroot string, filecontent []byte) error {
	var text = string(filecontent)
	if text == "" {
		text = `<ul>
			{{range .}}
				<li id="{{.Anchor}}">
					<a href="{{.URL}}">{{.Date}} {{.Title}}</a>
				</li>
			{{end}}
		</ul>`
	}

	return content.ParseWithData(
		t,
		text,
		func() []postPreview {
			return mb.previews
		},
	)
}

func readTmpl(t *template.Template, fsys fs.FS, filename string, defaultText string) *template.Template {
	var text = defaultText
	if fileText, err := fs.ReadFile(fsys, filename); err == nil {
		text = string(fileText)
	}
	t, _ = t.Clone()
	t.New("main").Parse(text)
	return t
}

// MakeHandler reads index.html and post.html (if exist) as "main" templates for index and post views.
func (mb *Miniblog) MakeHandler(fsys fs.FS, urlpath string, t *template.Template, contentFuncs map[string]seal.ContentFunc) http.Handler {
	indexTmpl := readTmpl(
		t,
		fsys,
		"index.html",
		`<ul>
			{{range .Previews}}
				<li id="{{.Anchor}}">
					<a href="{{.URL}}">{{.Date}} {{.Title}}</a>
				</li>
			{{end}}
		</ul>`,
	)

	postTmpl := readTmpl(
		t,
		fsys,
		"post.html",
		`<p><a href="{{.BackURL}}">Back to Blog</a></p>
		<p>{{.Date}}</p>
		{{template "post" .}}`,
	)

	var postFilenames []string
	entries, _ := fs.ReadDir(fsys, ".")
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) >= 10 && isoDate.MatchString(entry.Name()[:10]) {
			postFilenames = append(postFilenames, entry.Name())
		}
	}
	slices.Sort(postFilenames)
	slices.Reverse(postFilenames)

	var mux = http.NewServeMux()
	var previews []postPreview

	mux.HandleFunc("GET "+urlpath+"/", func(w http.ResponseWriter, r *http.Request) {
		indexTmpl.Execute(w, IndexData{
			TemplateData: seal.TemplateData{
				RequestURL: r.URL,
				URLPath:    urlpath,
			},
			Previews: previews,
		})
	})

	for _, fn := range postFilenames {
		contentFunc, ok := contentFuncs[path.Ext(fn)]
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
			tmpl.Execute(w, PostData{
				TemplateData: seal.TemplateData{
					RequestURL: r.URL,
					URLPath:    path.Join(urlpath, fileroot),
				},
				BackURL: urlpath + "#" + seal.MakeSlug(fileroot),
				Date:    date,
			},
			)
		})
	}

	mb.previews = previews

	return mux
}
