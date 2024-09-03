package models

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Store struct {
	Jobs                   []Job
	MaxConcurrentWorker    int
	MaxConcurrentWorkerJob int
	OutputFolder           string
}

type Job struct {
	DocumentId   string
	M3U8Link     string
	Initiator    string
	Files        []File
	IsCompleted  bool
	IsMerged     bool
	Status       string
	OutputFolder string
}

type File struct {
	Link       string
	Status     string
	FileIndex  int
	OutputPath string
}

type ChromeRequest struct {
	DocumentId string
	Url        string
	Initiator  string
}

func (s *Store) IsDocumentExists(documentId string) bool {
	for i := 0; i < len(s.Jobs); i++ {
		// fmt.Printf("checking: %s == %s\n", documentId, s.Jobs[i].DocumentId)
		if s.Jobs[i].DocumentId == documentId {
			return true
		}
	}

	return false
}

func (s *Store) AddJobContainer(request ChromeRequest) {
	fmt.Printf("added link: %s\n", request.Url)

	j := Job{
		DocumentId:   request.DocumentId,
		M3U8Link:     request.Url,
		Initiator:    request.Initiator,
		Files:        []File{},
		IsCompleted:  false,
		IsMerged:     false,
		Status:       "new",
		OutputFolder: time.Now().Format("20060102150405"),
	}

	// res, err := http.Get(link)
	buffer := downloadFile(j.M3U8Link, j.Initiator)
	if buffer == nil {
		j.Status = "failed"
	}

	scanner := bufio.NewScanner(buffer)

	var filePart int = 0

	baseURL := strings.Split(j.M3U8Link, "/")
	baseURL = baseURL[:len(baseURL)-1]
	basePath := strings.Join(baseURL, "/")

	for scanner.Scan() {
		line := scanner.Text()

		link := ""

		if len(line) > 4 && line[:4] == "http" {
			link = line
		} else {
			if strings.Contains(line, "EXTINF") {
				if scanner.Scan() {
					nextLine := scanner.Text()

					link = fmt.Sprintf("%s/%s", basePath, nextLine)
					if len(nextLine) > 4 && nextLine[:4] == "http" {
						link = nextLine
					}
				}
			}
		}

		if link != "" {
			j.Files = append(j.Files, File{
				Link:       link,
				Status:     "new",
				FileIndex:  filePart,
				OutputPath: fmt.Sprintf("%s/%s/%d", s.OutputFolder, j.OutputFolder, filePart),
			})

			filePart += 1
		}

	}

	// We skip links which doesn't have any job
	if filePart > 0 {
		s.Jobs = append(s.Jobs, j)
		s.StartDownload()
	}

}

func (s *Store) StartDownload() {
	// Find current active Jobs
	for i, job := range s.Jobs {
		if job.Status == "new" || job.Status == "active" {
			if !s.canStartDownload() {
				fmt.Printf("Currently at max download capacity: %d\n", s.MaxConcurrentWorker)
				return
			}

			s.Jobs[i].Status = "active"

			createFolder(fmt.Sprintf("%s/%s", s.OutputFolder, job.OutputFolder))
			fileChannel := make(chan *File, s.MaxConcurrentWorkerJob)

			for x := 0; x < s.MaxConcurrentWorkerJob; x++ {
				go worker(fileChannel, job.Initiator)
			}

			go func() {
				for j, file := range job.Files {
					if file.Status == "new" {
						file := &job.Files[j]
						fileChannel <- file
					}
				}
			}()
		}
	}
}

// Check if we reached the max number of downloaders
func (s *Store) canStartDownload() bool {
	activeJobs := 0
	for _, job := range s.Jobs {
		if job.Status == "active" {
			activeJobs += 1
		}
	}

	return activeJobs < s.MaxConcurrentWorker
}

func (s *Store) Save() {
	jsonString, err := json.Marshal(&s)

	if err != nil {
		fmt.Printf("failed to encode store into json string: %s\n", err)
		return
	}

	err = os.WriteFile("./store.json", jsonString, 0666)
	if err != nil {
		fmt.Printf("failed to save store into file: %s\n", err)
		return
	}
	// fmt.Println("Store Saved")
}

func (s *Store) Load() {
	cachedStore, err := os.ReadFile("./store.json")
	if err != nil {
		fmt.Printf("failed to read local store: %s", err)
		return
	}

	err = json.Unmarshal(cachedStore, &s)
	if err != nil {
		fmt.Printf("failed to unmarshal local store: %s", err)
		return
	}

	for i := range s.Jobs {
		if s.Jobs[i].Status == "active" {
			s.Jobs[i].Status = "new"
		}
	}

	s.StartDownload()
}

func (s *Store) GetProgress() string {
	output := ""
	for i, job := range s.Jobs {
		total := len(job.Files)
		success := 0

		for _, file := range job.Files {
			if file.Status == "success" {
				success += 1
			}
		}

		// Check if all file parts downloaded and is in active state
		if success == total && job.Status == "active" {
			s.Jobs[i].Status = "success"
			s.Jobs[i].IsCompleted = true

			s.StartDownload()
		}

		if s.Jobs[i].IsCompleted && !job.IsMerged && job.Status != "error" {
			if merge(s.OutputFolder, job) {
				s.Jobs[i].IsMerged = true

				os.RemoveAll(fmt.Sprintf("%s/%s", s.OutputFolder, job.OutputFolder))
			} else {
				s.Jobs[i].Status = "error"

				// We check the folder and see if all files are actually downloaded
				os.RemoveAll(fmt.Sprintf("%s/%s", s.OutputFolder, job.OutputFolder))
			}
		}

		percentage := float64(success) / float64(total)
		output += fmt.Sprintf("%d. (%s) %s: %d/%d (%.2f%%)\n", i+1, job.Status, job.OutputFolder, success, total, percentage*100)
	}

	return output
}

func (s *Store) Progress() {
	fmt.Println(s.GetProgress())
}

func merge(outputFolder string, job Job) bool {
	fmt.Printf("merging: %s", job.OutputFolder)
	mergeInstructionPath := fmt.Sprintf("%s/%s/%s.txt", outputFolder, job.OutputFolder, job.OutputFolder)
	fd, err := os.Create(mergeInstructionPath)

	if err != nil {
		fmt.Printf("failed to create merge instruction path: %s\n", err)
		return false
	}

	writer := bufio.NewWriter(fd)

	for _, file := range job.Files {
		writer.Write([]byte(fmt.Sprintf("file '%d'\n", file.FileIndex)))
		writer.Flush()
	}

	cmd := exec.Command("ffmpeg", "-f", "concat", "-i", mergeInstructionPath, "-c", "copy", "-bsf:a", "aac_adtstoasc", fmt.Sprintf("./%s/%s.mp4", outputFolder, job.OutputFolder))

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()

	if err != nil {
		fmt.Printf("Error merging files, err: %s\n", err)
		return false
	}

	return true
}

func createFolder(path string) {
	err := os.Mkdir(path, 0777)
	if err != nil {
		// fmt.Printf("failed to create folder, err: %s\n", err)
		return
	}
}

// Worker can take in job from anywhere
func worker(fileChan chan *File, initiator string) {
	for {
		select {
		case file := <-fileChan:
			// fmt.Printf("Worker: Downloading: %s\n", file.OutputPath)
			body := downloadFile(file.Link, initiator)

			if body == nil {
				return
			}

			outFile, _ := os.Create(file.OutputPath)
			_, _ = io.Copy(outFile, body)
			outFile.Close()

			file.Status = "success"
		}
	}
}

func downloadFile(url string, initiator string) io.Reader {
	// fmt.Printf("downloading: %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("failed to download file: %s", err)
		return nil
	}

	if initiator == "" {
		initiator = os.Getenv("INITIATOR")
	}

	req.Header = http.Header{
		"User-Agent":         {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"},
		"Referer":            {initiator},
		"Origin":             {fmt.Sprintf("%s/", initiator)},
		"Sec-Ch-Ua-Mobile":   {"?0"},
		"Sec-Ch-Ua-Platform": {"macOS"},
		"Sec-Fetch-Dest":     {"empty"},
		"Sec-Fetch-Mode":     {"cors"},
		"Sec-Fetch-Site":     {"cross-site"},
		"DNT":                {"1"},
		"Pragma":             {"no-cache"},
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("failed getting page: %s\n", url)
		return nil
	}

	return resp.Body
}
