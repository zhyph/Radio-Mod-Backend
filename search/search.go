package search

import (
	"fmt"
	"math"

	"github.com/raitonoberu/ytsearch"
)

type SearchResult struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Timestamp string `json:"timestamp"`
	Author    string `json:"author"`
	Ago       string `json:"ago"`
	Views     string `json:"views"`
	Seconds   int    `json:"seconds"`
}

func convertSecondsToTimestamp(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	seconds = seconds % 60

	var timestamp string

	if hours > 0 {
		timestamp = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	} else {
		timestamp = fmt.Sprintf("%d:%02d", minutes, seconds)
	}

	return timestamp
}

func formatViewCount(viewCount int) string {
	suffixes := []string{"", "K", "M", "B", "T"}
	log1000 := int(math.Floor(math.Log10(float64(viewCount)) / 3))

	if log1000 < 1 {
		return fmt.Sprintf("%d", viewCount)
	} else {
		rounded := fmt.Sprintf("%.1f", float64(viewCount)/math.Pow(10, float64(log1000*3)))
		return rounded + suffixes[log1000]
	}
}

func Search(query string) []SearchResult {
	var searchResults []SearchResult

	search := ytsearch.VideoSearch(query)
	result, err := search.Next()
	if err != nil {
		searchResults = append(searchResults, SearchResult{"tkzY_VwNIek", "Ween - Ocean Man", "2:08", "Ween", "6 years ago", "22M", 128})
		return searchResults
	}

	for _, video := range result.Videos {
		if video.Duration <= 0 {
			continue
		}
		searchResults = append(searchResults, SearchResult{
			video.ID,
			video.Title,
			convertSecondsToTimestamp(video.Duration),
			video.Channel.Title,
			video.PublishedTime,
			formatViewCount(video.ViewCount),
			video.Duration,
		})
	}

	if len(searchResults) == 0 {
		searchResults = append(searchResults, SearchResult{"tkzY_VwNIek", "Ween - Ocean Man", "2:08", "Ween", "6 years ago", "22M", 128})
	}

	return searchResults
}
