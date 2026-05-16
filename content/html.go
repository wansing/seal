package content

import (
	"html/template"
	"text/template/parse"
)

// Html parses the filecontent as an html template using Golang's html/template package.
func HTML(t *template.Template, urlpath, fileroot string, filecontent []byte) error {
	parsed, err := t.Parse(string(filecontent)) // $parsed is only required for post-processing
	if err != nil {
		return err
	}

	// Post-Processing: Make relative href and src paths absolute.
	// This must be made before template execution, because execution brings templates with different urlpaths together.
	// We modify the TextNodes (which contain the HTML code) of the parsed template. Other templates defined in filecontent are not modified.
	if parsed != nil && parsed.Tree != nil && parsed.Tree.Root != nil {
		for _, node := range parsed.Tree.Root.Nodes {
			if node.Type() == parse.NodeText { // TextNodes can't be parsed because they are not well-formed, but can be tokenized
				newNodeText := AbsHrefSrc(node.String(), urlpath)
				node.(*parse.TextNode).Text = []byte(newNodeText)
			}
		}
	}
	return nil
}
