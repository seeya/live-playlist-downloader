package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/seeya/live-playlist-downloader/downloader/models"
)

func Default(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Downloader"))
}

func DownloadM3U8(store *models.Store) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			io.WriteString(res, "failed to read request")
			return
		}

		var payload map[string]string
		err = json.Unmarshal(body, &payload)
		if err != nil {
			io.WriteString(res, "failed to unmarshal request")
			return
		}

		store.AddJobContainer(payload["url"])
		io.WriteString(res, "ok")
	}
}
