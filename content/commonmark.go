package content

import (
	"bytes"
	"html/template"
	"regexp"

	"github.com/wansing/seal"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var commonmark = goldmark.New(
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
	goldmark.WithExtensions(
		extension.NewFootnote(),
		extension.NewLinkify(),
		extension.NewTypographer(),
	),
)

var (
	templateExpr = regexp.MustCompile(`\{([a-z-]{1,32})\}`)
	templateRepl = `{{template "$1" .}}`
)

// Commonmark parses the filecontent as CommonMark Markdown and calls Html on the result.
func Commonmark(t *template.Template, urlpath, fileroot string, filecontent []byte, broker *seal.Broker) error {
	var buf bytes.Buffer
	if err := commonmark.Convert(filecontent, &buf); err != nil {
		return err
	}
	htmlcontent := templateExpr.ReplaceAll(buf.Bytes(), []byte(templateRepl))
	return HTML(t, urlpath, fileroot, htmlcontent, broker)
}
