package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	discord "com.radio/discord"
	download "com.radio/download"
	search "com.radio/search"
	"github.com/gin-gonic/gin"
)

var (
	config      Config
	downloadDir = ""
)

type Config struct {
	Endpoint         string   `json:"endpoint"`
	Port             string   `json:"port"`
	UseProxies       bool     `json:"useProxies"`
	Proxies          []string `json:"proxies"`
	Retries          int      `json:"retries"`
	UseFallback      bool     `json:"useFallback"`
	BannedTerms      []string `json:"bannedTerms"`
	BannedPlayfabIDs []string `json:"bannedPlayfabIDs"`
}

func DefaultConfig() *Config {
	return &Config{
		Endpoint:         "http://172.217.22.14",
		Port:             "3045",
		UseProxies:       false,
		Proxies:          []string{"localhost", "http://proxy1.example.com:4444", "http://proxy2.example.com:4444"},
		Retries:          1,
		UseFallback:      true,
		BannedTerms:      []string{"fart", "idiot"},
		BannedPlayfabIDs: []string{"playfabId01", "playfabId02"},
	}
}

func postSearch(c *gin.Context) {
	result := make(chan gin.H)
	go func(context *gin.Context) {
		var requestData struct {
			SearchString string `json:"searchString"`
		}

		if err := c.BindJSON(&requestData); err != nil {
			c.JSON(http.StatusOK, gin.H{"valid": false})
			result <- gin.H{"valid": false}
			return
		}

		searchResults := search.Search(requestData.SearchString)
		result <- gin.H{"valid": true, "results": searchResults}
	}(c.Copy())

	c.JSON(http.StatusOK, <-result)
}

func stringInSlice(target string, list *[]string) bool {
	for _, item := range *list {
		if item == target {
			return true
		}
	}
	return false
}

func filterPlayfabID(playfabId string) bool {
	return stringInSlice(playfabId, &config.BannedPlayfabIDs)
}

func filterVideoTitle(title string) bool {
	lowerTitle := strings.ToLower(title)
	inputWords := strings.Fields(lowerTitle)
	for _, word := range inputWords {
		if stringInSlice(word, &config.BannedTerms) {
			return true
		}
	}

	return false
}

func postQueue(c *gin.Context) {
	result := make(chan gin.H)
	go func(context *gin.Context) {
		var requestData struct {
			VideoId       string  `json:"videoId"`
			VideoTitle    string  `json:"videoTitle"`
			ServerName    string  `json:"server"`
			PlayfabId     string  `json:"playfabId"`
			PlayerName    *string `json:"playerName,omitempty"`
			ServerWebhook *string `json:"serverWebhook,omitempty"`
		}

		if err := c.BindJSON(&requestData); err != nil {
			fmt.Println("\x1b[31mQueue request binding failed.\x1b[0m")
			result <- gin.H{"valid": false}
			return
		}

		if filterPlayfabID(requestData.PlayfabId) {
			fmt.Println("\x1b[38;5;171mBlocked Playfab ID: " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]\x1b[0m")
			result <- gin.H{"valid": false}
			return
		}

		if filterVideoTitle(requestData.VideoTitle) {
			fmt.Println("\x1b[38;5;214mBlocked Video Title: " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]\x1b[0m")
			result <- gin.H{"valid": false}
			return
		}

		downloadResult := download.DownloadResult{}

		for i := 0; i < config.Retries+1; i++ {
			downloadResult = download.Download(requestData.VideoId, config.UseProxies, &config.Proxies)
			if downloadResult.Error != "" {
				if strings.Contains(downloadResult.Error, "can't bypass age restriction") {
					break
				}
				fmt.Println("Retry:", i+1)
				continue
			}
			break
		}

		if downloadResult.Valid {
			fmt.Println("Success: " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]")
		} else if config.UseFallback {
			fmt.Println("\x1b[31mFailed (reverting to fallback): " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]" + " ~ " + downloadResult.Error + " ~ " + downloadResult.Proxy + "\x1b[0m")
			downloadResult = download.DownloadFallback(requestData.VideoId, config.UseProxies, &config.Proxies)
			if downloadResult.Valid {
				fmt.Println("Success (fallback): " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]")
			} else {
				fmt.Println("\x1b[31mFailed (fallback): " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]" + " ~ " + downloadResult.Error + " ~ " + downloadResult.Proxy + "\x1b[0m")
			}
		} else {
			fmt.Println("\x1b[31mFailed: " + requestData.VideoTitle + " ~ " + requestData.ServerName + " ~ [" + requestData.PlayfabId + "]" + " ~ " + downloadResult.Error + " ~ " + downloadResult.Proxy + "\x1b[0m")
		}

		if downloadResult.Valid && requestData.ServerWebhook != nil {
			go discord.Webhooks(requestData.PlayfabId, *requestData.PlayerName, requestData.VideoTitle, requestData.VideoId, requestData.ServerName, *requestData.ServerWebhook)
		}

		uuid := fmt.Sprintf("%s:%s/%s", config.Endpoint, config.Port, downloadResult.Uuid)

		result <- gin.H{"valid": downloadResult.Valid, "videoId": requestData.VideoId, "uuid": uuid,
			"maxRes": downloadResult.MaxRes, "videoTitle": requestData.VideoTitle}
	}(c.Copy())

	c.JSON(http.StatusOK, <-result)
}

func postPlaylist(c *gin.Context) {
	playlistId := c.GetHeader("playlistId")
	fmt.Println("Playlist Id:", playlistId)

	if playlistId == "" {
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}

	url := "http://radiomod.thesaltyseacow.com:3065/playlist"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}

	req.Header.Set("playlistId", playlistId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Request failed. Response:", resp.Status)
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}

	var responseData map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&responseData); err != nil {
		fmt.Println("Decode failed.")
		c.JSON(http.StatusOK, gin.H{"valid": false})
		return
	}

	c.JSON(http.StatusOK, responseData)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST")
		c.Next()
	}
}

func LoadConfig(file string) (bool, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		config = *DefaultConfig()
		content, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return false, err
		}
		err = os.WriteFile(file, content, 0644)
		if err != nil {
			return false, err
		}
		return false, nil
	} else {
		content, err := os.ReadFile(file)
		if err != nil {
			return true, err
		}
		err = json.Unmarshal(content, &config)
		if err != nil {
			return true, err
		}
		config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")
		return true, nil
	}
}

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	configExists, configErr := LoadConfig(filepath.Join(exeDir, "config.json"))
	if configErr != nil {
		fmt.Println("Error loading config:", configErr)
		return
	}
	if !configExists {
		fmt.Println("config.json has been created! Please edit it and then restart the radio backend.")
		return
	}

	songHistoryPath := filepath.Join(exeDir, "songHistory.json")

	downloadDir = filepath.Join(exeDir, "downloaded-videos")

	_, err = os.Stat(downloadDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(downloadDir, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return
		}
	} else if err != nil {
		fmt.Println("Error checking directory:", err)
		return
	}

	download.LoadSongHistory(downloadDir, songHistoryPath)
	download.StartCheckLoop()
	ginMode := "release"
	gin.SetMode(ginMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Static("/", downloadDir)
	router.POST("/search", postSearch)
	router.POST("/queue", postQueue)
	router.POST("/playlist", postPlaylist)
	router.Use(corsMiddleware())
	fmt.Println("Radio Mod backend running on port", config.Port)
	router.Run(":" + config.Port)
}
