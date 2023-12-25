package templates

import (
	"fmt"
	"html/template"
)

// Copy returns a clone of t where the parse tree of the template named src is added as dst, overwriting any previous dst template.
func Copy(t *template.Template, dst, src string) (*template.Template, error) {
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
