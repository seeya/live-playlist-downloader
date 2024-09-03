package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/seeya/live-playlist-downloader/handlers"
	"github.com/seeya/live-playlist-downloader/models"
)

func main() {
	store := &models.Store{
		Jobs:                   []models.Job{},
		MaxConcurrentWorker:    3,
		MaxConcurrentWorkerJob: 15,
		OutputFolder:           "./output",
	}

	store.Load()

	store.MaxConcurrentWorker = 3
	store.MaxConcurrentWorkerJob = 15

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
	router.HandleFunc("GET /health", handlers.Default)
	router.HandleFunc("GET /progress", handlers.GetProgress(store))
	router.HandleFunc("POST /", handlers.DownloadM3U8(store))
	// Static File Server
	router.Handle("GET /", handlers.StaticFileServer(store))

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
