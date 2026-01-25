package handlers

import (
	"bytes"
	"html/template"
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
	var tokenizer = html.NewTokenizerFragment(bytes.NewReader(htm), "body")
	tokenizer.SetMaxBuf(4096) // roughly the maximum number of bytes tokenized

	var bytesRead = 0
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

		bytesRead += len(tokenizer.Raw())
		if bytesRead > 4000 {
			break
		}
	}

	return ""
}
