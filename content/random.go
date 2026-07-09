package content

import (
	"html/template"
	"math/rand"
	"strings"
)

func RandomHTML(t *template.Template, urlpath, fileroot string, filecontent []byte) error {
	var options []string
	for line := range strings.Lines(string(filecontent)) {
		options = append(options, AbsHrefSrc(line, urlpath))
	}
	return ParseWithData(
		t,
		"{{.}}",
		func() template.HTML {
			var o string
			if len(options) > 0 {
				o = options[rand.Intn(len(options))]
			}
			return template.HTML(o)
		},
	)
}
