package pkg

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const addMemeUsage = "```\nUsage: !addmeme <command-name> - creates a command that plays an audio file\n\n" +
	"This command can only be used as a reply to a message which contains a single wav file attachment. " +
	"The attachment file name must end in .wav, and the command cannot include spaces or eomji's.```"

func (c *Command) AddMeme() {
	if c.MessageEvent.ReferencedMessage == nil {
		SendMessageWithError(c.Session, c.MessageEvent, addMemeUsage, "failed to send usage for add-meme")
		return
	}

	if c.MessageEvent.ReferencedMessage.Attachments == nil || len(c.MessageEvent.ReferencedMessage.Attachments) != 1 {
		SendMessageWithError(c.Session, c.MessageEvent, addMemeUsage, "failed to send usage for add-meme")
		return
	}

	attachment := c.MessageEvent.ReferencedMessage.Attachments[0]
	if !strings.HasSuffix(attachment.Filename, ".wav") {
		SendMessageWithError(c.Session, c.MessageEvent, addMemeUsage, "failed to send usage for add-meme")
	}

	if strings.Contains(c.Opts.Content, " ") {
		SendMessageWithError(c.Session, c.MessageEvent, addMemeUsage, "failed to send usage for add-meme")
		return
	}

	req, err := http.NewRequest(http.MethodGet, attachment.URL, nil)
	if err != nil {
		SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("failed to download referenced audio file: %v", err), "failed to send usage for add-meme")
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("failed to download referenced audio file: %v", err), "failed to send usage for add-meme")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("failed to download referenced audio file, received unexpected response code %d", resp.StatusCode), "failed to send usage for add-meme")
		return
	}

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("failed to download referenced audio file, encountered error reading response body: %v", err), "failed to send usage for add-meme")
		return
	}

	file, err := os.OpenFile(fmt.Sprintf("%s/%s.wav", MemeLocation, c.Opts.Content), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("failed to download referenced audio file, encountered error creating file: %v", err), "failed to send usage for add-meme")
		return
	}

	_, err = file.Write(fileBytes)
	if err != nil {
		SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("failed to download referenced audio file, encountered error writing file: %v", err), "failed to send usage for add-meme")
		return
	}

	SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("File downloaded, command '!%s' will be created shortly", c.Opts.Content), "failed to send usage for add-meme")
}
