package seal

import "html/template"

// A ContentFunc populates a template from a given file content.
type ContentFunc func(dirpath string, filecontent []byte, tmpl *template.Template) error