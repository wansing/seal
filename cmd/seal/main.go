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

	var s seal.Seal
	s = seal.Seal{
		Fsys: os.DirFS("."),
		FileExts: map[string]seal.TemplateGen{
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

	if !isatty.IsTerminal(os.Stdout.Fd()) {
		s.Filenames["git-update"] = s.GitUpdateHandler
	}

	s.ListenAndServe("127.0.0.1:" + strconv.Itoa(httpPort))
}
