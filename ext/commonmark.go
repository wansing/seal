package ext

import "gitlab.com/golang-commonmark/markdown"

var commonmark = markdown.New(markdown.HTML(true), markdown.Linkify(true), markdown.Typographer(true), markdown.MaxNesting(10))

func Commonmark(input []byte) string {
	return commonmark.RenderToString(input)
}
