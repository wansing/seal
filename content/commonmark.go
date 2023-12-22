package content

import (
	"regexp"

	"github.com/wansing/seal"
	"gitlab.com/golang-commonmark/markdown"
)

var commonmark = markdown.New(markdown.HTML(true), markdown.Linkify(true), markdown.Typographer(true), markdown.MaxNesting(10))

var templateCmd = regexp.MustCompile(`\{([a-z-]{1,32})\}`)

// Commonmark parses the filecontent as CommonMark Markdown and calls Html on the result.
// Use {name} to execute a template.
func Commonmark(dir *seal.Dir, filestem string, filecontent []byte) error {
	s := commonmark.RenderToString(filecontent)
	s = templateCmd.ReplaceAllString(s, `{{template "$1" .}}`)
	return Html(dir, filestem, []byte(s))
}
