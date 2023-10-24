package main

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/wansing/seal"
	"github.com/wansing/seal/ext"
)

func main() {
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = "127.0.0.1:8080"
	}

	var srv *seal.Server
	srv = &seal.Server{
		Conf: seal.Config{
			Fsys: os.DirFS("."),
			Content: map[string]seal.ContentFunc{
				".countdown": ext.Countdown,
				".html":      ext.Html,
				".md":        ext.Commonmark,
			},
			Handlers: map[string]seal.HandlerGen{
				"redirect": seal.Redirect,
				"update":   srv.UpdateHandler,
			},
		},
	}

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		srv.Conf.Handlers["git-update"] = srv.GitUpdateHandler
	}

	srv.ListenAndServe(listen)
}
