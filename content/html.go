package content

import (
	"html/template"
	"path"
	"strings"
	"text/template/parse"

	"github.com/wansing/seal"
	"golang.org/x/net/html"
)

// Html parses the filecontent as an html template using Golang's html/template package.
//
// The template variable $dir is set to the URLPath of the dir where the content file is located.
func HTML(t *template.Template, urlpath, fileroot string, filecontent []byte, broker *seal.Broker) error {
	parsed, err := t.Parse(string(filecontent)) // $parsed is only required for post-processing
	if err != nil {
		return err
	}

	// Post-Processing: Make relative href and src paths absolute.
	// This must be made before template execution, because execution brings templates with different urlpaths together.
	// We modify the TextNodes (which contain the HTML code) of the parsed template. Other templates defined in filecontent are not modified.
	if parsed == nil || parsed.Tree == nil || parsed.Tree.Root == nil {
		return nil
	}
	var contextTag string
	for _, node := range parsed.Tree.Root.Nodes {
		if node.Type() != parse.NodeText {
			continue
		}

		tokenizer := html.NewTokenizerFragment(strings.NewReader(node.String()), contextTag) // TextNodes can't be parsed because they are not well-formed, but can be tokenized
		var newNodeText strings.Builder
		for {
			tokenType := tokenizer.Next()
			if tokenType == html.ErrorToken {
				break // assuming tokenizer.Err() == io.EOF
			}
			if tokenType != html.StartTagToken {
				newNodeText.Write(tokenizer.Raw()) // raw copy everything except start tags
				continue
			}

			token := tokenizer.Token()
			contextTag = token.Data
			for i, a := range token.Attr {
				if (a.Key == "src" || a.Key == "href") && !path.IsAbs(a.Val) { // TODO ToLower?
					token.Attr[i].Val = path.Join(urlpath, a.Val)
				}
			}
			newNodeText.WriteString(token.String())
		}
		node.(*parse.TextNode).Text = []byte(newNodeText.String())
	}

	return nil
}
