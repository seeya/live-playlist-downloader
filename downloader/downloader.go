package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/seeya/live-playlist-downloader/downloader/handlers"
	"github.com/seeya/live-playlist-downloader/downloader/models"
)

func main() {
	store := &models.Store{
		Jobs:                   []models.Job{},
		MaxConcurrentWorker:    3,
		MaxConcurrentWorkerJob: 15,
		OutputFolder:           "./output",
	}

	store.Load()

	go func() {
		for range time.Tick(time.Second * 10) {
			store.Save()
		}
	}()

	go func() {
		for range time.Tick(time.Second * 5) {
			store.Progress()
		}
	}()

	router := http.NewServeMux()
	router.HandleFunc("GET /", handlers.Default)
	router.HandleFunc("POST /", handlers.DownloadM3U8(store))

	server := http.Server{
		Addr:    ":3002",
		Handler: router,
	}

	fmt.Println("Listening on port: 3002")
	err := server.ListenAndServe()

	if err != nil {
		panic(err)
	}

	fmt.Println("Server closed")
}
