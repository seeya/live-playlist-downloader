package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/seeya/live-playlist-downloader/models"
)

func Default(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("OK"))
}

func DownloadM3U8(store *models.Store) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			io.WriteString(res, "failed to read request")
			return
		}

		var payload models.ChromeRequest
		err = json.Unmarshal(body, &payload)
		if err != nil {
			io.WriteString(res, "failed to unmarshal request")
			return
		}

		fmt.Printf("Link: %s\nInitiator: %s\nDocumentId: %s\n\n", payload.Url, payload.Initiator, payload.DocumentId)
		if store.IsDocumentExists(payload.DocumentId) {
			fmt.Println("Link already exists, skipping")
		} else {
			store.AddJobContainer(payload)
		}

		io.WriteString(res, "ok")
	}
}

func GetProgress(store *models.Store) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte(store.GetProgress()))
	}
}

func StaticFileServer(store *models.Store) http.Handler {
	return http.FileServer(http.Dir(store.OutputFolder))
}
