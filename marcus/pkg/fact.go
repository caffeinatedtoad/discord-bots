package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Command) SayFact() {
	resp, err := http.Get("https://uselessfacts.jsph.pl/api/v2/facts/random")
	if err != nil {
		_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("the fact API returned an unexpected error: %v", err))
		return
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("failed to read fact response: %v", err))
		return
	}

	resp.Body.Close()

	fact := FactResponse{}
	err = json.Unmarshal(b, &fact)
	if err != nil {
		_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("failed to unmarshal fact response: %v", err))
		return
	}

	go c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("```\n%s\n```", fact.Text))
	GetAndSpeak(c.Session, c.MessageEvent, fact.Text, c.Opts.ChannelName)
}

// getting free facts from https://uselessfacts.jsph.pl/api/v2/facts/random

type FactResponse struct {
	Id        string `json:"id"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	SourceUrl string `json:"source_url"`
	Language  string `json:"language"`
	Permalink string `json:"permalink"`
}
