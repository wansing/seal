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

	otherRepo := seal.MakeDirRepo(config, "../other-repo")

	rootRepo := seal.MakeDirRepo(config, ".")
	rootRepo.Conf.Handlers["other-repo"] = func(dir *seal.Dir, filestem string, filecontent []byte) seal.Handler {
		if err := otherRepo.Update(dir); err != nil {
			log.Printf("error updating other repo: %v", err)
		}
		return otherRepo.Serve
	}

	srv := &seal.Server{
		Repository: rootRepo,
	}
	if err := srv.Repository.Update(nil); err != nil {
		log.Fatalf("error updating root repo: %v", err)
	}

	log.Printf("listening to %s", listen)
	http.HandleFunc("/update", srv.UpdateHandler("change-me"))
	http.HandleFunc("/git-update-other", otherRepo.GitUpdateHandler("change-me", srv))
	http.HandleFunc("/git-update-root", rootRepo.GitUpdateHandler("change-me", srv))
	http.Handle("/", srv)
	http.ListenAndServe(listen, nil)
}
