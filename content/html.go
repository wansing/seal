package content

import (
	"github.com/wansing/seal"
)

// Html parses the filecontent as an html template using Golang's templating engine.
//
// The template variable $dir is set to the URLPath of the dir where the content file is located.
func Html(dir *seal.Dir, filestem string, filecontent []byte) error {
	// We want to create links and embed images which are relative to a directory.
	//
	// Rewriting <a href="..."> and <img src="..."> is hard because:
	// Execution puts templates from different directories together.
	// So we can only manipulate them before execution, leaving two options:
	// Parsing as HTML would be hard because the renderer would escape quotes in template actions.
	// Modifying the template parse tree would also be hard because it does not parse HTML.
	//
	// Instead, let's pass the directory to the template through a variable $dir.
	// We have to inject the template action before parsing, else parsing fails when trying to access $dir.
	// This does only work for the main template, not for {{define}}'d templates.
	htm := `{{$dir := "` + dir.URLPath + `"}}` + string(filecontent)

	_, err := dir.Template.New(filestem).Parse(htm)
	return err
}
