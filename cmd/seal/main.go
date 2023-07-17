package main

import (
	"os"

	"github.com/wansing/seal"
	"github.com/wansing/seal/ext"
)

func main() {
	var s seal.Seal
	s = seal.Seal{
		Fsys: os.DirFS("."),
		Exts: map[string]seal.Ext{
			".html": ext.Html,
			".md":   ext.Commonmark,
		},
		Filenames: map[string]seal.HandlerGen{
			"redirect": seal.Redirect,
			"update":   s.UpdateHandler,
		},
		Params: map[string]seal.Handler{
			// TODO
		},
	}

	s.ListenAndServe("127.0.0.1:8081")
}
