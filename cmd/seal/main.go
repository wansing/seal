package main

import (
	"os"
	"strconv"

	"github.com/mattn/go-isatty"
	"github.com/wansing/seal"
	"github.com/wansing/seal/ext"
)

func main() {
	httpPort, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil || httpPort < 1 || httpPort > 65535 {
		httpPort = 8080
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

	srv.ListenAndServe("127.0.0.1:" + strconv.Itoa(httpPort))
}
