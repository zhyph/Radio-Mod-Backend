package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Embed struct {
	Description string `json:"description"`
	Color       int    `json:"color"`
	Footer      struct {
		Text string `json:"text"`
	} `json:"footer"`
}

type Message struct {
	Content     interface{} `json:"content"`
	Embeds      []Embed     `json:"embeds"`
	Attachments []string    `json:"attachments"`
	Flags       int         `json:"flags"`
}

func Webhooks(playfabId string, playerName string, videoTitle string, videoUrl string, serverName string, serverWebhook string) {
	message := Message{
		Content: nil,
		Embeds: []Embed{
			{
				Description: fmt.Sprintf("**[%s] %s** - [%s](%s%s)", playfabId, playerName, videoTitle, "https://www.youtube.com/watch?v=", videoUrl),
				Color:       5439356,
				Footer: struct {
					Text string `json:"text"`
				}{
					Text: serverName,
				},
			},
		},
		Attachments: []string{},
		Flags:       4096,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	globalWebhook := "http://radiomod.thesaltyseacow.com:4573/webhook"

	resp, err := http.Post(globalWebhook, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if len(serverWebhook) > 0 {
		resp, err := http.Post(serverWebhook, "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()
	}
}
