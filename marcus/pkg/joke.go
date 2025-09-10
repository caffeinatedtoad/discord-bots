package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Command) SayJoke() {
	resp, err := http.Get("https://v2.jokeapi.dev/joke/Miscellaneous,Dark?blacklistFlags=nsfw,religious,political,racist,sexist,explicit&type=single")
	if err != nil {
		_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("the joke API returned an unexpected error: %v", err))
		return
	}

	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("the joke API returned an unexpected response: %v", err))
		return
	}

	j := Joke{}

	err = json.Unmarshal(b, &j)
	if err != nil {
		_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("the joke API returned an unexpected response: %v", err))
		return
	}

	intro := fmt.Sprintf("Heres a \"%s\" joke:", j.Category)
	if j.Category == "Misc" {
		intro = "Heres a miscellaneous joke:"
	}

	joke := fmt.Sprintf("%s %s", intro, strings.ReplaceAll(j.Joke, "\n", " "))
	go c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("```\n%s\n```", joke))
	GetAndSpeak(c.Session, c.MessageEvent, joke, c.Opts.ChannelName)
}

type Joke struct {
	Error    bool   `json:"error"`
	Category string `json:"category"`
	Type     string `json:"type"`
	Joke     string `json:"joke"`
	Flags    struct {
		Nsfw      bool `json:"nsfw"`
		Religious bool `json:"religious"`
		Political bool `json:"political"`
		Racist    bool `json:"racist"`
		Sexist    bool `json:"sexist"`
		Explicit  bool `json:"explicit"`
	} `json:"flags"`
	Id   int    `json:"id"`
	Safe bool   `json:"safe"`
	Lang string `json:"lang"`
}
