package main

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wansing/seal"
	"github.com/wansing/seal/content"
)

var testFS = fstest.MapFS{
	"favicon.ico": &fstest.MapFile{
		Data: []byte("ICON"),
	},
	"html.html": &fstest.MapFile{
		Data: []byte(`<html><body><main>{{block "main" .}}{{end}}</main></body></html>`),
	},
	"main.md": &fstest.MapFile{
		Data: []byte(`# Hello`),
	},
	"site/main.html": &fstest.MapFile{
		Data: []byte(`<h1><a href="{{$dir}}">Site</a>{{block "site" .}}{{end}}</h1>`),
	},
	"site/subsite/site.md": &fstest.MapFile{
		Data: []byte(`## Subsite`),
	},
	"redirect-absolute-url/redirect": &fstest.MapFile{
		Data: []byte("https://example.com"),
	},
	"redirect-absolute-path/redirect": &fstest.MapFile{
		Data: []byte("/path"),
	},
	"redirect-relative-path/redirect": &fstest.MapFile{
		Data: []byte("path"),
	},
	"nested-definitions/foo.html": &fstest.MapFile{
		Data: []byte(`This is ignored. {{define "main"}}This is main.{{end}}`),
	},
	"empty-dir": &fstest.MapFile{
		Mode: fs.ModeDir,
	},
	"other/other-repo": &fstest.MapFile{},
}

var otherFS = fstest.MapFS{
	"main.md": &fstest.MapFile{
		Data: []byte(`# Other repository`),
	},
}

var repo = &seal.Repository{
	Fsys: testFS,
}

var otherRepo = &seal.Repository{
	Fsys: otherFS,
}

var srv = &seal.Server{
	Content: map[string]seal.ContentFunc{
		".html": content.Html,
		".md":   content.Commonmark,
	},
	Handlers: map[string]seal.HandlerGen{
		"redirect": seal.RedirectHandler,
	},
	Repo: repo,
}

func init() {
	srv.Handlers["other-repo"] = func(dir *seal.Dir, filestem string, filecontent []byte) (seal.Handler, error) {
		var errs = []seal.Error{}
		_ = srv.ReloadRepo(otherRepo, dir, &errs)
		return otherRepo.Serve, nil
	}
}

func TestSeal(t *testing.T) {
	if err := srv.Reload(); err != nil {
		t.Fatal(err)
	}
	go http.ListenAndServe("127.0.0.1:8081", srv)

	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		input string
		want  string
	}{
		{input: "/", want: `<html><body><main><h1>Hello</h1>
</main></body></html>`},
		{input: "/favicon.ico", want: "ICON"},
		{input: "/site", want: `<html><body><main><h1><a href="/site">Site</a></h1></main></body></html>`},
		{input: "/site/subsite", want: `<html><body><main><h1><a href="/site">Site</a><h2>Subsite</h2>
</h1></main></body></html>`},
		{input: "/redirect-absolute-url", want: `<a href="https://example.com">See Other</a>.`},
		{input: "/redirect-absolute-path", want: `<a href="/path">See Other</a>.`},
		{input: "/redirect-relative-path", want: `<a href="/redirect-relative-path/path">See Other</a>.`},
		{input: "/nested-definitions", want: `<html><body><main>This is main.</main></body></html>`},
		{input: "/empty-dir", want: `404 page not found`},
		{input: "/other", want: `<html><body><main><h1>Other repository</h1>
</main></body></html>`},
	}

	// don't follow redirects
	http.DefaultClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	for _, test := range tests {
		resp, err := http.DefaultClient.Get("http://127.0.0.1:8081" + test.input)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(got)) != test.want {
			t.Fatalf("%s: expected: %v, got: %v", test.input, test.want, string(got))
		}
	}
}

func TestReload(t *testing.T) {

	testFS["main.md"] = &fstest.MapFile{
		Data: []byte(`# Reloaded`),
	}

	srv.Reload()
	time.Sleep(100 * time.Millisecond)

	input := "/"
	want := `<html><body><main><h1>Reloaded</h1>
</main></body></html>`

	resp, err := http.DefaultClient.Get("http://127.0.0.1:8081" + input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != want {
		t.Fatalf("%s: expected: %v, got: %v", input, want, string(got))
	}
}
