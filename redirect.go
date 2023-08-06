package seal

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

func Redirect(filecontent []byte) Handler {
	redir := strings.TrimSpace(string(filecontent))
	var join = false
	if uri, err := url.Parse(redir); err == nil {
		// uri is "relative (a path, without a host) or absolute (starting with a scheme)"
		if !uri.IsAbs() && !path.IsAbs(uri.Path) {
			join = true
		}
	}
	return func(dir *Dir, reqpath []string, w http.ResponseWriter, r *http.Request) {
		redir := redir // don't touch original value
		if join {
			redir = path.Join(r.URL.Path, redir) // handler is called only when the Dir is targeted directly, so r.URL.Path should be the path to this Dir
		}
		http.Redirect(w, r, redir, http.StatusSeeOther)
	}
}
