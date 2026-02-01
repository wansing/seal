package content

import (
	"html/template"
	"regexp"

	"github.com/wansing/seal"
	"gitlab.com/golang-commonmark/markdown"
)

var commonmark = markdown.New(markdown.HTML(true), markdown.Linkify(true), markdown.Typographer(true), markdown.MaxNesting(10))

var templateCmd = regexp.MustCompile(`\{([a-z-]{1,32})\}`)

// Commonmark parses the filecontent as CommonMark Markdown and calls Html on the result.
func Commonmark(t *template.Template, urlpath, fileroot string, filecontent []byte, broker *seal.Broker) error {
	htmlcontent := commonmark.RenderToString(filecontent)
	htmlcontent = templateCmd.ReplaceAllString(htmlcontent, `{{template "$1" .}}`)
	return HTML(t, urlpath, fileroot, []byte(htmlcontent), broker)
}
