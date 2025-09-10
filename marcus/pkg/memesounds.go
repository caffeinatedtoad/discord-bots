package pkg

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
)

var memeFiles = map[string]string{}

func init() {
	memeFiles = make(map[string]string)
	err := filepath.Walk("memes", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, "wav") {
			memeFiles[info.Name()] = path
		}

		return nil
	})
	if err != nil {
		slog.Error("couldn't read meme files: %v", err)
	}
}

func (c *Command) Mets() {
	if files == nil || len(files) == 0 {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, "Failed to initialize meme files.")
		if err != nil {
			c.Logger.Error(fmt.Sprintf("ERR: %v", err))
		}
		return
	}

	SpeakFile(c.Session, c.MessageEvent, memeFiles["mets.wav"], c.Opts.ChannelName)
}
