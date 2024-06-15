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
	"time"
)

type Store struct {
	Jobs                   []Job
	MaxConcurrentWorker    int
	MaxConcurrentWorkerJob int
	OutputFolder           string
}

type Job struct {
	M3U8Link     string
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

func (s *Store) AddJobContainer(link string) {
	fmt.Printf("got link! %s\n", link)

	j := Job{
		M3U8Link:     link,
		Files:        []File{},
		IsCompleted:  false,
		IsMerged:     false,
		Status:       "new",
		OutputFolder: time.Now().Format("20060102150405"),
	}

	res, err := http.Get(link)
	if err != nil {
		j.Status = "failed"
	}

	scanner := bufio.NewScanner(res.Body)

	var filePart int = 0

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 4 && line[:4] == "http" {
			j.Files = append(j.Files, File{
				Link:       line,
				Status:     "new",
				FileIndex:  filePart,
				OutputPath: fmt.Sprintf("%s/%s/%d", s.OutputFolder, j.OutputFolder, filePart),
			})

			filePart += 1
		} else {

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

			for i := 0; i < s.MaxConcurrentWorkerJob; i++ {
				go worker(fileChannel)
			}

			go func() {
				for i, file := range job.Files {
					if file.Status == "new" {
						file := &job.Files[i]
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

	s.StartDownload()
}

func (s *Store) Progress() {
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

		if s.Jobs[i].IsCompleted && !job.IsMerged {
			if merge(s.OutputFolder, job) {
				s.Jobs[i].IsMerged = true

				os.RemoveAll(fmt.Sprintf("%s/%s", s.OutputFolder, job.OutputFolder))
			} else {
				s.Jobs[i].Status = "error"
			}
		}

		percentage := float64(success) / float64(total)
		output += fmt.Sprintf("%d. (%s) %s: %d/%d (%.2f%%)\n", i+1, job.Status, job.OutputFolder, success, total, percentage*100)
	}

	fmt.Println(output)
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
func worker(fileChan chan *File) {
	for {
		select {
		case file := <-fileChan:
			// fmt.Printf("Worker: Downloading: %s\n", file.OutputPath)
			body := downloadFile(file.Link)

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

func downloadFile(url string) io.Reader {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("failed to download file: %s", err)
		return nil
	}

	req.Header = http.Header{
		"user-agent": {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"},
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("failed getting page: %s\n", url)
		return nil
	}

	return resp.Body
}
