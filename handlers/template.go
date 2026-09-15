package handlers

import (
	"bytes"
	"html/template"
	"io"
	"strings"
	"text/template/parse"

	"golang.org/x/net/html"
)

func Heading(t *template.Template) string {
	for _, node := range t.Tree.Root.Nodes {
		if node.Type() == parse.NodeText {
			if h := heading(node.(*parse.TextNode).Text); h != "" {
				return h
			}
		}
	}
	return ""
}

func heading(htm []byte) string {
	r := &io.LimitedReader{
		R: bytes.NewReader(htm),
		N: 4096,
	}
	var tokenizer = html.NewTokenizerFragment(r, "body")
	var headingTag = ""
	var result = &strings.Builder{}

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break // assuming tokenizer.Err() == io.EOF
		}

		tagNameBytes, _ := tokenizer.TagName()
		tagName := string(tagNameBytes)

		if headingTag == "" {
			if tt == html.StartTagToken && (tagName == "h1" || tagName == "h2" || tagName == "h3" || tagName == "h4") {
				headingTag = tagName
			}
		} else {
			if tt == html.EndTagToken && tagName == headingTag {
				return result.String()
			}
			result.Write(tokenizer.Raw())
		}
	}

	return result.String()
}
