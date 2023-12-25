package templates

import (
	"fmt"
	"html/template"
)

// Rename returns a clone of t where the template src is renamed to dst, overwriting any previous dst template.
func Rename(t *template.Template, dst, src string) (*template.Template, error) {
	srcTemplate := t.Lookup(src)
	if srcTemplate == nil {
		return nil, fmt.Errorf(`template "%s" not defined`, src)
	}
	t, err := t.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning template: %w", err)
	}
	if _, err := t.AddParseTree(dst, srcTemplate.Tree); err != nil {
		return nil, fmt.Errorf("adding parse tree: %w", err)
	}
	return t, nil
}
