package seal

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type HandlerGen func(dir *Dir, filestem string, filecontent []byte) Handler

// A Handler responds to an HTTP request. It must return true if execution shall continue, false to stop execution.
type Handler func(reqpath []string, w http.ResponseWriter, r *http.Request) bool

// MakeRedirectHandler returns a Handler which does a HTTP 303 redirect if the (remaining) request path is empty.
// The redirect target is taken from the file content. If the target is relative, the handler will prepend the request URL path.
func MakeRedirectHandler(_ *Dir, _ string, filecontent []byte) Handler {
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
	}
}

// MakeTemplateHandler returns a Handler which executes the "html" template from dir.Template if reqpath is empty.
// If an error is returned, a template with an error message is executed.
func MakeTemplateHandler(dir *Dir, _ string, _ []byte) Handler {
	return func(reqpath []string, w http.ResponseWriter, r *http.Request) bool {
		if len(reqpath) > 0 {
			return true
		}

		if dir.Template != nil {
			var buf bytes.Buffer
			err := dir.Template.ExecuteTemplate(&buf, "html", struct {
				Dir     *Dir
				Request *http.Request
			}{
				dir,
				r,
			})
			if err != nil {
				buf.Reset()
				errExecuteTemplate.Execute(&buf, err)
			}
			io.Copy(w, &buf)
		}
		return false
	}
}
