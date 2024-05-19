package seal

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
)

type HandlerGen func(dir *Dir, filestem string, filecontent []byte) (Handler, error)

// A Handler responds to an HTTP request. It must return true if execution shall continue, false to stop execution.
type Handler func(reqpath []string, w http.ResponseWriter, r *http.Request) bool

// RedirectHandler returns a Handler which does a HTTP 303 redirect if the (remaining) request path is empty.
// The redirect target is taken from the file content. If the target is relative, the handler will prepend the request URL path.
func RedirectHandler(_ *Dir, _ string, filecontent []byte) (Handler, error) {
	redir := strings.TrimSpace(string(filecontent))
	var join = false
	if uri, err := url.Parse(redir); err == nil {
		// uri is "relative (a path, without a host) or absolute (starting with a scheme)"
		if !uri.IsAbs() && !path.IsAbs(uri.Path) {
			join = true
		}
	}

	return func(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
		if len(reqpath) > 0 {
			return true
		}

		redir := redir // don't touch original value
		if join {
			redir = path.Join(r.URL.Path, redir) // handler is called only when the Dir is targeted directly, so r.URL.Path should be the path to this Dir
		}
		http.Redirect(w, r, redir, http.StatusSeeOther)
		return false
	}, nil
}

// TemplateHandler returns a Handler which, if reqpath is empty, executes dir.Template with TemplateData.
func TemplateHandler(dir *Dir, _ string, _ []byte) (Handler, error) {
	if dir.Template == nil {
		return nil, errors.New("no template")
	}

	// test template execution, clone before so dir.Template can be extended later
	t, err := dir.Template.Clone()
	if err != nil {
		return nil, err
	}
	if err := t.Execute(io.Discard, TemplateData{
		Dir:     dir,
		Request: httptest.NewRequest(http.MethodGet, dir.URLPath, nil), // target parameter must not contain spaces
	}); err != nil {
		return nil, err
	}

	return func(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
		if len(reqpath) > 0 {
			return true
		}
		dir.Template.Execute(w, TemplateData{
			Dir:     dir,
			Request: r,
		}) // ignore error, assume that initial execution test was enough
		return false
	}, nil
}

type TemplateData struct {
	Dir     *Dir
	Request *http.Request
}

// ExecuteTemplate executes the named template from dir.Template (not from data.Dir).
// Use this function to embed content of a specific Dir, e.g. a blog post preview.
func (data TemplateData) ExecuteTemplate(dir *Dir, name string) (template.HTML, error) {
	var buf bytes.Buffer
	err := dir.Template.ExecuteTemplate(&buf, name, TemplateData{
		Dir:     dir,
		Request: data.Request,
	})
	return template.HTML(buf.String()), err
}
