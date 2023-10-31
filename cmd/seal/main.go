package main

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/wansing/seal"
	"github.com/wansing/seal/content"
	"github.com/wansing/shiftpad/ical"
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
				".calendar-bs5": content.CalendarBS5{
					FeedCache: &ical.FeedCache{
						Config: ical.MustLoadConfig(".calendar.json"),
					},
				}.Handle,
				".countdown": content.Countdown,
				".html":      content.Html,
				".md":        content.Commonmark,
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
