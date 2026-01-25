package main

import (
	"log"
	"net/http"
	"os"

	"github.com/wansing/seal"
	"github.com/wansing/seal/content"
	"github.com/wansing/seal/handlers/miniblog"
)

func main() {
	listen := "127.0.0.1:8080"
	reloadSecret := "change-me"

	content := map[string]seal.ContentFunc{
		".calendar-bs5": content.CalendarBS5{}.Make,
		".countdown":    content.Countdown,
		".html":         content.Html,
		".md":           content.Commonmark,
	}

	srv := &seal.Server{
		FS:      os.DirFS("."),
		Content: content,
		Handlers: map[string]seal.HandlerGen{
			".blog": miniblog.Make,
		},
	}
	srv.Reload()

	log.Printf("listening to %s", listen)
	http.HandleFunc("/errors", srv.ErrorsHandler())
	http.HandleFunc("/reload", seal.ReloadHandler(reloadSecret, srv.Reload))
	http.HandleFunc("/git-reload", seal.GitReloadHandler(reloadSecret, ".", srv.Reload))
	http.Handle("/", srv) // least precendece
	http.ListenAndServe(listen, nil)
}
