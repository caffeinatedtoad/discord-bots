package pkg

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"marcus/pkg/util"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

var Memes *MemeSet
var MemeLocation string

type MemeSet struct {
	sync.Mutex
	Memes map[string]*memeCollection
}

type memeCollection struct {
	Name  string
	Files []memeFile `json:"files"`

	child *memeCollection
}

type memeFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetTopLevelFiles walks the meme directory and creates individual
// commands for each file
func GetTopLevelFiles(dirName string) error {
	files, err := os.ReadDir(dirName)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.TrimSuffix(file.Name(), ".wav")
		Memes.Memes[name] = &memeCollection{
			Name: name,
			Files: []memeFile{
				{
					Name: name,
					Path: dirName + "/" + file.Name(),
				},
			},
		}
	}

	return err
}

func (m *memeCollection) walkWavFiles(toplevel bool, workingDirectory, name string) error {
	slog.Debug(fmt.Sprintf("walking wave files for %s", name))
	Memes.Memes[name] = m

	files, err := os.ReadDir(workingDirectory)
	if err != nil {
		slog.Error("couldn't read meme files: %v", err)
		return nil
	}

	for _, info := range files {
		if info.IsDir() {
			slog.Debug(fmt.Sprintf("processing directory %s", info.Name()))
			m.child = &memeCollection{
				Name: name,
			}

			newName := workingDirectory + "-" + strings.TrimSuffix(info.Name(), ".wav")
			if toplevel {
				newName = strings.TrimSuffix(info.Name(), ".wav")
			}

			err = m.child.walkWavFiles(false, workingDirectory+"/"+info.Name(), newName)
			if err != nil {
				slog.Error(fmt.Sprintf("couldn't read meme files: %v", err))
			}
			continue
		}

		if strings.HasSuffix(info.Name(), ".wav") {
			fileName := strings.TrimSuffix(info.Name(), ".wav")
			m.Files = append(m.Files, memeFile{
				Name: fileName,
				Path: workingDirectory + "/" + info.Name(),
			})
		}
	}

	return nil

}

func init() {
	Memes = &MemeSet{
		Memes: make(map[string]*memeCollection),
	}

	MemeLocation = os.Getenv("MEMES_LOCATION")
	if MemeLocation == "" {
		MemeLocation = "memes"
	}
	_, err := os.Stat(MemeLocation)
	if err != nil {
		slog.Warn(fmt.Sprintf("memes directory not found: %v, meme commands disabled", err))
		return
	}

	if err = GetTopLevelFiles(MemeLocation); err != nil {
		slog.Error("couldn't read meme files: %v", err)
		return
	}

	memes := &memeCollection{
		Name: "",
	}

	err = memes.walkWavFiles(true, MemeLocation, "")
	if err != nil {
		slog.Error(fmt.Sprintf("couldn't read meme files: %v", err))
	}

	slog.Info("loaded memes")
	j, _ := json.MarshalIndent(Memes, "", " ")
	slog.Info(string(j))

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				Memes.Lock()
				slog.Debug("refreshing memes")
				if err = GetTopLevelFiles(MemeLocation); err != nil {
					slog.Error("couldn't read meme files: %v", err)
					return
				}

				memes := &memeCollection{
					Name: "",
				}

				err = memes.walkWavFiles(true, MemeLocation, "")
				if err != nil {
					slog.Error(fmt.Sprintf("couldn't read meme files: %v", err))
				}
				Memes.Unlock()
				slog.Debug("refreshed memes")
			}
		}
	}()
}

func (c *Command) Meme(collection *memeCollection) {
	Memes.Lock()
	defer Memes.Unlock()

	if len(collection.Files) == 1 {
		c.TTS.SpeakFile(c.Session, c.MessageEvent, collection.Files[0].Path, c.TTSOpts.ChannelName)
		return
	}

	if c.SubcommandString != "" {
		found := false
		for _, file := range collection.Files {
			if file.Name == c.SubcommandString {
				found = true
				c.TTS.SpeakFile(c.Session, c.MessageEvent, file.Path, c.TTSOpts.ChannelName)
				break
			}
		}

		if !found {
			util.SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("Couldn't find a meme with the name %s", collection.Name+"-"+c.SubcommandString), "Couldn't find a meme with the name")
			return
		}

		return
	}

	rand.Intn(len(collection.Files))
	fileName := collection.Files[rand.Intn(len(collection.Files))]

	c.TTS.SpeakFile(c.Session, c.MessageEvent, fileName.Path, c.TTSOpts.ChannelName)
}

func (c *Command) ListMemes() {
	Memes.Lock()
	defer Memes.Unlock()

	var x strings.Builder
	x.WriteString("```")
	for k, v := range Memes.Memes {
		for _, file := range v.Files {
			if k == "" {
				x.WriteString(fmt.Sprintf("%s\n", file.Name))
				continue
			}
			x.WriteString(fmt.Sprintf("%s-%s\n", k, file.Name))
		}
	}
	x.WriteString("```")

	util.SendMessageWithError(c.Session, c.MessageEvent, x.String(), "failed to list memes")
}
