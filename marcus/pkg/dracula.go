package pkg

import (
	"fmt"
	"io/fs"
	"log/slog"
	"math/rand"
	"path/filepath"
	"strings"
)

var files []string

func init() {
	err := filepath.Walk("dracula", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, "wav") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		slog.Error("couldn't read dracula files: %v", err)
	}
}

func (c *Command) Dracula() {
	if files == nil {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, "Failed to initialize dracula voice files.")
		if err != nil {
			c.Logger.Error(fmt.Sprintf("ERR: %v", err))
		}
		return
	}

	file := files[rand.Intn(len(files))]
	SpeakFile(c.Session, c.MessageEvent, file, c.Opts.ChannelName)
}
