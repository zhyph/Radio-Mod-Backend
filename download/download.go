package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kkdai/youtube/v2"
	"github.com/wader/goutubedl"
)

type HistoryEntry struct {
	Uuid      string    `json:"uuid"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	currentProxyIndex = 0
	songHistory       []HistoryEntry
	songHistoryPath   = ""
	downloadDir       = ""
)

func checkStatusCode(url string) bool {
	response, err := http.Get(url)
	if err != nil {
		return false
	}
	defer response.Body.Close()

	return response.StatusCode == http.StatusOK
}

func addHistoryEntry(uuid string) {
	newEntry := HistoryEntry{
		Uuid:      uuid,
		Timestamp: time.Now(),
	}

	songHistory = append(songHistory, newEntry)
	saveSongHistory()
}

func saveSongHistory() {
	jsonData, err := json.Marshal(songHistory)
	if err != nil {
		return
	}

	err = os.WriteFile(songHistoryPath, jsonData, 0644)
	if err != nil {
		return
	}
}

func LoadSongHistory(downloadPath string, songHistoryPth string) {
	downloadDir = downloadPath
	songHistoryPath = songHistoryPth

	jsonData, err := os.ReadFile(songHistoryPath)
	if err != nil {
		return
	}

	err = json.Unmarshal(jsonData, &songHistory)
	if err != nil {
		return
	}
}

func DeleteFile(filepath string) error {
	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = os.Remove(filepath)
	if err != nil {
		return err
	}

	return nil
}

func checkSongHistory() {
	currentTime := time.Now()
	var filteredData []HistoryEntry
	for _, item := range songHistory {
		if currentTime.Sub(item.Timestamp) <= time.Hour {
			filteredData = append(filteredData, item)
		} else {
			DeleteFile(filepath.Join(downloadDir, item.Uuid))
		}
	}
	songHistory = filteredData
	saveSongHistory()
}

func StartCheckLoop() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			checkSongHistory()
		}
	}()
}

type DownloadResult struct {
	Valid  bool   `json:"valid"`
	Uuid   string `json:"uuid"`
	MaxRes bool   `json:"maxRes"`
	Proxy  string `json:"proxy"`
	Error  string `json:"error"`
}

func rotateProxy(list *[]string) string {
	proxyUrl := (*list)[currentProxyIndex]
	currentProxyIndex = (currentProxyIndex + 1) % len(*list)
	fmt.Println("Proxy:", proxyUrl)
	return proxyUrl
}

func Download(videoID string, useProxies bool, proxies *[]string) DownloadResult {
	returnResult := DownloadResult{Valid: false, Uuid: "", MaxRes: false, Proxy: "", Error: ""}

	var httpClient *http.Client

	if useProxies {
		proxyUrl := rotateProxy(proxies)
		returnResult.Proxy = proxyUrl
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: func(r *http.Request) (*url.URL, error) {
					if proxyUrl == "localhost" {
						return nil, nil
					}
					return url.Parse(proxyUrl)
				},
			},
		}
	} else {
		httpClient = &http.Client{}
	}

	client := youtube.Client{HTTPClient: httpClient}

	video, err := client.GetVideo(videoID)
	if err != nil {
		returnResult.Error = err.Error()
		return returnResult
	}

	extension := ".webm"
	formats := video.Formats.Type("audio/webm")
	if len(formats) == 0 {
		formats = video.Formats.Type("audio/mp4")
		extension = ".m4a"
	}
	if len(formats) == 0 {
		fmt.Println("format length is STILL zero")
		return returnResult
	}

	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		returnResult.Error = err.Error()
		return returnResult
	}
	defer stream.Close()

	uuid := uuid.NewString() + extension

	file, err := os.Create(filepath.Join(downloadDir, uuid))
	if err != nil {
		returnResult.Error = err.Error()
		DeleteFile(filepath.Join(downloadDir, uuid))
		return returnResult
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		returnResult.Error = err.Error()
		DeleteFile(filepath.Join(downloadDir, uuid))
		return returnResult
	}

	returnResult.Valid = true
	returnResult.Uuid = uuid
	returnResult.MaxRes = checkStatusCode("https://img.youtube.com/vi/" + videoID + "/maxresdefault.jpg")
	addHistoryEntry(uuid)
	return returnResult
}

func DownloadFallback(videoID string, useProxies bool, proxies *[]string, cookiesPath string) DownloadResult {
	returnResult := DownloadResult{Valid: false, Uuid: "", MaxRes: false, Proxy: "", Error: ""}

	var result goutubedl.Result
	var goutubeErr error
	goutubedl.Path = "yt-dlp"

	if useProxies {
		proxyUrl := rotateProxy(proxies)
		returnResult.Proxy = proxyUrl
		result, goutubeErr = goutubedl.New(context.Background(), videoID, goutubedl.Options{ProxyUrl: proxyUrl, Cookies: cookiesPath})
	} else {
		result, goutubeErr = goutubedl.New(context.Background(), videoID, goutubedl.Options{Cookies: cookiesPath})
	}

	if goutubeErr != nil {
		returnResult.Error = goutubeErr.Error()
		return returnResult
	}
	downloadResult, err := result.Download(context.Background(), "bestaudio")
	if err != nil {
		returnResult.Error = err.Error()
		return returnResult
	}
	defer downloadResult.Close()

	uuid := uuid.NewString() + ".webm"

	f, err := os.Create(filepath.Join(downloadDir, uuid))
	if err != nil {
		returnResult.Error = err.Error()
		return returnResult
	}
	defer f.Close()
	io.Copy(f, downloadResult)

	returnResult.Valid = true
	returnResult.Uuid = uuid
	returnResult.MaxRes = checkStatusCode("https://img.youtube.com/vi/" + videoID + "/maxresdefault.jpg")
	addHistoryEntry(uuid)
	return returnResult
}
