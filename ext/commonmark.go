package ext

import (
	"html/template"
	"regexp"

	"gitlab.com/golang-commonmark/markdown"
)

var commonmark = markdown.New(markdown.HTML(true), markdown.Linkify(true), markdown.Typographer(true), markdown.MaxNesting(10))

var templateCmd = regexp.MustCompile("\\{([a-z-]{1,32})\\}")

func Commonmark(dirpath string, input []byte, tmpl *template.Template) error {
	s := commonmark.RenderToString(input)
	s = templateCmd.ReplaceAllString(s, `{{template "$1" .}}`)
	return Html(dirpath, []byte(s), tmpl)
}
