package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func merge(folder string, totalFiles int) {
	linksPath := fmt.Sprintf("./downloads/%s/mylist.txt", folder)
	fd, err := os.Create(linksPath)
	defer os.RemoveAll(linksPath)

	if err != nil {
		fmt.Printf("failed to create file, err: %s\n", err)
		return
	}

	writer := bufio.NewWriter(fd)

	for i := 1; i <= totalFiles; i++ {
		writer.Write([]byte(fmt.Sprintf("file '%d'\n", i+1)))
		writer.Flush()
	}

	cmd := exec.Command("ffmpeg", "-f", "concat", "-i", fmt.Sprintf("./downloads/%s/mylist.txt", folder), "-c", "copy", "-bsf:a", "aac_adtstoasc", fmt.Sprintf("./downloads/%s/video.mp4", folder))

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()

	fmt.Printf("Output: %q\n", out.String())

	if err != nil {
		fmt.Printf("Error merging files, err: %s\n", err)
		return
	}

	cleanup(fmt.Sprintf("./downloads/%s", folder))
	// ffmpeg -f concat -i mylist.txt -c copy -bsf:a aac_adtstoasc video.mp4
}

// keep only mp4 file
func cleanup(folder string) {
	dir, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("failed to read dir, err: %s\n", err)
		return
	}

	for _, file := range dir {
		if !strings.Contains(file.Name(), ".mp4") && !strings.Contains(file.Name(), ".txt") {
			os.Remove(fmt.Sprintf("%s/%s", folder, file.Name()))
		}
	}
}

func download(job Job, folder string) {
	resp, err := http.Get(job.Link)

	if err != nil {
		fmt.Printf("Error downloading %s, err: %s\n", job.Link, err)
	}

	outFile, _ := os.Create(fmt.Sprintf("./downloads/%s/%d", folder, job.Index))
	defer outFile.Close()

	_, _ = io.Copy(outFile, resp.Body)
}

func worker(jobs chan Job, folder string) {
	go func() {
		for {
			select {
			case job := <-jobs:
				download(job, folder)
			}
		}
	}()
}

func downloadList(url string) io.Reader {
	resp, err := http.Get(url)

	if err != nil {
	}

	return resp.Body
}

const (
	MAX_WORKERS = 15
)

type UrlRequest struct {
	Url string `json:"url"`
}

type Job struct {
	Index int
	Link  string
}

func doDownload(link string) {
	contents := downloadList(link)

	scanner := bufio.NewScanner(contents)

	jobs := make(chan Job, MAX_WORKERS)

	folder := time.Now().Format("20060102150405")

	// showName := "A-J"
	// folder = strings.Split(link, showName)[1]
	// folder = strings.Split(folder, "/")[0]
	// folder = fmt.Sprintf("%s%s", showName, folder)

	err := os.Mkdir(fmt.Sprintf("./downloads/%s", folder), 0777)
	if err != nil {
		fmt.Printf("failed to create folder, err: %s\n", err)
		return
	}

	os.WriteFile(fmt.Sprintf("./downloads/%s/link.txt", folder), []byte(link), 0777)

	for i := 0; i < MAX_WORKERS; i++ {
		go worker(jobs, folder)
	}

	progress := 0

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "https") {
			progress += 1
			fmt.Printf("Count: %d\n", progress)
			jobs <- Job{Index: progress, Link: line}
		}
	}

	fmt.Printf("Jobs done %d\n", progress)
	merge(folder, progress)
}

func removeElement(slice []string, index int) []string {

	// Append function used to append elements to a slice
	// first parameter as the slice to which the elements
	// are to be added/appended second parameter is the
	// element(s) to be appended into the slice
	// return value as a slice
	return append(slice[:index], slice[index+1:]...)
}

func main() {
	AUTO_DOWNLOAD := false
	AUTO_DOWNLOAD_FILTER := ".m3u8"
	AUTO_DOWNLOAD_FILTER_IGNORE := ""
	link := os.Args[1]

	var lists []string
	lists = append(lists, "https://github.com/seeya")

	if link == "server" {
		app := fiber.New()

		app.Get("/", func(c *fiber.Ctx) error {
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return c.SendFile("./index.html")
		})

		app.Get("/list", func(c *fiber.Ctx) error {
			return c.JSON(lists)
		})

		app.Post("/", func(c *fiber.Ctx) error {
			var urlReq UrlRequest
			err := c.BodyParser(&urlReq)
			if err != nil {
				fmt.Printf("failed to parse body, err: %s\n", err)
				return c.SendString("failed to parse body")
			}

			for _, val := range lists {
				if val == urlReq.Url {
					return c.SendString("already in list")
				}
			}

			fmt.Printf("%s\n", urlReq.Url)

			if AUTO_DOWNLOAD {
				if strings.Contains(urlReq.Url, AUTO_DOWNLOAD_FILTER) {
					if !strings.Contains(urlReq.Url, AUTO_DOWNLOAD_FILTER_IGNORE) {
						go doDownload(urlReq.Url)
					}
				}
			} else {
				lists = append(lists, urlReq.Url)
			}

			return c.SendString("ok")
		})

		app.Post("/download", func(c *fiber.Ctx) error {
			var urlReq UrlRequest
			err := c.BodyParser(&urlReq)
			if err != nil {
				fmt.Printf("failed to parse body, err: %s\n", err)
				return c.SendString("failed to parse body")
			}

			go doDownload(urlReq.Url)

			for i, val := range lists {
				if val == urlReq.Url {
					lists = removeElement(lists, i)
					return c.SendString("removed from list")
				}
			}

			return c.SendString("downloading")
		})

		app.Listen(":3000")
	} else if link == "merge" {
		total, _ := strconv.Atoi(os.Args[3])
		merge(os.Args[2], total)
		return
	}

	doDownload(link)
}
