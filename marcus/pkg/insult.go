package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func (c *Command) SayInsult() {
	resp, err := http.Get("https://evilinsult.com/generate_insult.php?lang=en&type=json")
	if err != nil {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("The insult API returned an error: %v", err))
		if err != nil {
			log.Println(fmt.Sprintf("ERR: %v", err))
		}
		return
	}

	rb := InsultResponse{}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("The insult API returned an unexpected response: %v", err))
		if err != nil {
			log.Println(fmt.Sprintf("ERR: %v", err))
		}
		return
	}
	defer resp.Body.Close()

	err = json.Unmarshal(b, &rb)
	if err != nil {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("The insult API returned an unexpected response: %v", err))
		if err != nil {
			log.Println(fmt.Sprintf("ERR: %v", err))
		}
		return
	}

	go c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("```\n%s\n```", rb.Insult))
	GetAndSpeak(c.Session, c.MessageEvent, rb.Insult, c.Opts.ChannelName)
}

// getting insults from https://evilinsult.com/generate_insult.php?lang=en&type=json

type InsultResponse struct {
	Number    string `json:"number"`
	Language  string `json:"language"`
	Insult    string `json:"insult"`
	Created   string `json:"created"`
	Shown     string `json:"shown"`
	Createdby string `json:"createdby"`
	Active    string `json:"active"`
	Comment   string `json:"comment"`
}
