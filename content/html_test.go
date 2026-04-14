package content

import (
	"bytes"
	"html/template"
	"testing"
)

func TestHrefSrc(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// add urlpath
		{`<a href="bar">Example</a>`, `<a href="/foo/bar">Example</a>`},
		{`<img src="image.html">`, `<img src="/foo/image.html">`},
		// keep absolute paths
		{`<a href="/bar">Example</a>`, `<a href="/bar">Example</a>`},
		{`<img src="/image.html">`, `<img src="/image.html">`},
		// keep fragment links
		{`<a href="#foo">Example</a>`, `<a href="#foo">Example</a>`},
		// keep full urls
		{`<a href="https://example.com">Example</a>`, `<a href="https://example.com">Example</a>`},
		{`<img src="https://example.com">`, `<img src="https://example.com">`},
	}

	for _, test := range tests {
		tmpl := template.New("html")
		err := HTML(tmpl, "/foo", "main", []byte(test.input))
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		tmpl.Execute(&buf, nil)
		got := buf.String()
		if got != test.want {
			t.Fatalf("got %s, want %s", got, test.want)
		}
	}
}

func TestTemplateNesting(t *testing.T) {
	tmpl := template.Must(template.New("html").Parse(`<html><body>{{template "main" .}}</body></html>`))
	err := HTML(tmpl.New("main"), "/foo", "main", []byte(`<p>Hello World!</p><a href="bar"><img src="image.jpg"></a>`))
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
