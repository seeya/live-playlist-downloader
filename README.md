# Live Playlist Downloader

## Introduction

This project attempts to download individual video files obtained from live streaming endpoints and consolidate into a `video.mp4` file using `ffmpeg`.

Having the video in a `.mp4` container is easier to be played on media players like `VLC`.

## Usage 1 - Download the file immediately

```
go run main.go "https://link.com/index.m3u8"
```

## Usage 2 - Web Server + Chrome Extension

This will spin up a HTTP server listening on port 3000. Together with the chrome extension, it will attempt to filter only requests which match the pattern defined in at `content.js#L12`
The link will automatically popup at `http://localhost:3000`.
Clicking the link will start the download.

```
go run main.go server
```

## Dependencies

1. ffmpeg
2. golang
