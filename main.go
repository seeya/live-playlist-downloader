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

func merge(folder string) {
	dir, err := os.ReadDir(fmt.Sprintf("./downloads/%s", folder))

	if err != nil {
		panic(err)
	}

	linksPath := fmt.Sprintf("./downloads/%s/mylist.txt", folder)
	fd, err := os.Create(linksPath)
	defer os.RemoveAll(linksPath)

	if err != nil {
		panic(err)
	}

	writer := bufio.NewWriter(fd)

	/*
	   The file name should be in the format seg-1-f1-v1-a1.ts
	   We first create a string array of size len(dir)
	   Then we loop through the dir and split the file name by - to obtain its index
	   We then put the file name in the array at the index - 1
	   Once done, we loop through the array and write the file name to the mylist.txt file
	*/

	fileList := make([]string, len(dir))
	for _, file := range dir {
		splitted := strings.Split(file.Name(), "-")

		if len(splitted) == 1 {
			continue
		}

		index, err := strconv.Atoi(splitted[1])

		if err != nil {
			panic(err)
		}

		if index-1 < len(fileList) {
			fileList[index-1] = file.Name()
		}
	}

	for _, file := range fileList {
		if file != "" {
			writer.Write([]byte(fmt.Sprintf("file '%s'\n", file)))
			writer.Flush()
		}
	}

	cmd := exec.Command("ffmpeg", "-f", "concat", "-i", fmt.Sprintf("./downloads/%s/mylist.txt", folder), "-c", "copy", "-bsf:a", "aac_adtstoasc", fmt.Sprintf("./downloads/%s/video.mp4", folder))

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()

	fmt.Printf("Output: %q\n", out.String())

	if err != nil {
		fmt.Printf("Error merging files, err: %s\n", err)
		panic(err)
	}

	cleanup(fmt.Sprintf("./downloads/%s", folder))
	// ffmpeg -f concat -i mylist.txt -c copy -bsf:a aac_adtstoasc video.mp4
}

// keep only mp4 file
func cleanup(folder string) {
	dir, err := os.ReadDir(folder)
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		if !strings.Contains(file.Name(), ".mp4") && !strings.Contains(file.Name(), ".txt") {
			os.Remove(fmt.Sprintf("%s/%s", folder, file.Name()))
		}
	}
}

func download(url string, folder string) {
	resp, err := http.Get(url)

	if err != nil {
		fmt.Printf("Error downloading %s, err: %s\n", url, err)
	}

	index := strings.Index(url, ".ts")

	if index == -1 {
		fmt.Printf("No found name found in link %s\n", url)
	}

	foundStartOfFileName := false
	cursor := index

	for foundStartOfFileName == false {
		cursor -= 1

		if url[cursor] == '/' {
			foundStartOfFileName = true
		}
	}

	filename := url[cursor : index+3]
	outFile, err := os.Create(fmt.Sprintf("./downloads/%s/%s", folder, filename))
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
}

func worker(jobs chan string, folder string) {
	go func() {
		for {
			select {
			case link := <-jobs:
				download(link, folder)
			}
		}
	}()
}

func downloadList(url string) io.Reader {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	return resp.Body
}

const (
	MAX_WORKERS = 15
)

type UrlRequest struct {
	Url string `json:"url"`
}

func doDownload(link string) {
	contents := downloadList(link)

	scanner := bufio.NewScanner(contents)

	jobs := make(chan string, MAX_WORKERS)

	folder := time.Now().Format("20060102150405")

	err := os.Mkdir(fmt.Sprintf("./downloads/%s", folder), 0777)
	if err != nil {
		panic(err)
	}

	for i := 0; i < MAX_WORKERS; i++ {
		go worker(jobs, folder)
	}

	progress := 0

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "https") {
			progress += 1
			fmt.Printf("Count: %d\n", progress)
			jobs <- line
		}
	}

	fmt.Printf("Jobs done %d\n", progress)
	merge(folder)
}

func main() {
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

			fmt.Printf("%s\n", urlReq.Url)

			lists = append(lists, urlReq.Url)

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

			return c.SendString("downloading")
		})

		app.Listen(":3000")
	}

	doDownload(link)
}
