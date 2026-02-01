package content

import (
	"bytes"
	"html/template"
	"testing"
)

func TestHTML(t *testing.T) {
	tmpl := template.Must(template.New("html").Parse(`<html><body>{{template "main" .}}</body></html>`))
	err := HTML(tmpl.New("main"), "/foo", "main", []byte(`<p>Hello World!</p><a href="bar"><img src="image.jpg"></a>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, nil)
	got := buf.String()
	want := `<html><body><p>Hello World!</p><a href="/foo/bar"><img src="/foo/image.jpg"></a></body></html>`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
