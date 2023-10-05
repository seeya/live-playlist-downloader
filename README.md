# Live Playlist Downloader

## Introduction

This project attempts to download individual video files obtained from live streaming endpoints and consolidate into a `video.mp4` file using `ffmpeg`.

Having the video in a `.mp4` container is easier to be played on media players like `VLC`.

## Usage

```
go run main.go "https://link.com/index.m3u8"
```

## Dependencies

1. ffmpeg
2. golang
