package main

import (
	"log"
	"net/http"
	"os"

	"github.com/wansing/seal"
	"github.com/wansing/seal/content"
)

func main() {
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = "127.0.0.1:8080"
	}

	config := seal.Config{
		Content: map[string]seal.ContentFunc{
			".calendar-bs5": content.CalendarBS5{}.Parse,
			".countdown":    content.Countdown,
			".html":         content.Html,
			".md":           content.Commonmark,
		},
		Handlers: map[string]seal.HandlerGen{
			"redirect": seal.MakeRedirectHandler,
		},
	}

	rootRepo := seal.MakeDirRepository(config, ".")

	srv := &seal.Server{
		Repo: rootRepo,
	}
	if err := srv.Update(); err != nil {
		log.Fatalf("error updating root repo: %v", err)
	}

	log.Printf("listening to %s", listen)
	http.HandleFunc("/errors", srv.ErrorsHandler())
	http.HandleFunc("/update", srv.UpdateHandler("change-me"))
	http.HandleFunc("/git-update-root", rootRepo.GitUpdateHandler("change-me", srv))
	http.Handle("/", srv)
	http.ListenAndServe(listen, nil)
}
