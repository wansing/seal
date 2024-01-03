package main

import (
	"crypto/rand"
	"encoding/base64"
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
	secret := os.Getenv("SECRET")
	if secret == "" {
		var bs = make([]byte, 16)
		if _, err := rand.Read(bs); err != nil {
			log.Fatalf("error making random secret: %v", err)
		}
		secret = base64.RawURLEncoding.EncodeToString(bs)
		log.Printf("generated temporary reload secret: %s", secret)
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
	if err := srv.Reload(); err != nil {
		log.Fatalf("error loading root repo: %v", err)
	}

	log.Printf("listening to %s", listen)
	http.HandleFunc("/errors", srv.ErrorsHandler())
	http.HandleFunc("/reload", srv.ReloadHandler(secret))
	http.HandleFunc("/git-reload-root", rootRepo.GitReloadHandler(secret, srv))
	http.Handle("/", srv)
	http.ListenAndServe(listen, nil)
}
