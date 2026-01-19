package pkg

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type MemeSet struct {
	*sync.Map
}

type MemeHit struct {
	IsDir bool
	Path  string
}

func (m *MemeSet) BuildMemeSet() error {
	MemeLocation := os.Getenv("MEMES_LOCATION")
	if MemeLocation == "" {
		MemeLocation = "memes"
	}
	return bm(MemeLocation, MemeLocation, m)
}

func (m *MemeSet) ListMemes() string {
	var memes []string
	m.Range(func(key, value interface{}) bool {
		memes = append(memes, key.(string))
		return true
	})
	slices.Sort(memes)
	return strings.Join(memes, "```\n```")
}

func (m *MemeSet) GetMeme(command string) (string, bool) {
	v, ok := m.Load(command)
	if !ok {
		fmt.Println("meme not found")
		return "", false
	}

	hit, ok := v.(MemeHit)
	if !ok {
		return "", false
	}

	if hit.IsDir {
		entries, err := os.ReadDir(hit.Path)
		if err != nil {
			fmt.Println("failed to read directory", err)
			return "", false
		}
		for i := 0; i < 10; i++ {
			randomEntry := entries[rand.Intn(len(entries))]
			stat, err := os.Stat(filepath.Join(hit.Path, randomEntry.Name()))
			if err != nil {
				fmt.Println("failed to stat file", err)
				return "", false
			}
			// don't return a directory if we can
			if stat.IsDir() {
				fmt.Println("found directory when randomly selecting, ignoring")
				continue
			}

			return filepath.Join(hit.Path, randomEntry.Name()), true
		}
	}

	return hit.Path, true
}

func bm(base, root string, fin *MemeSet) error {
	return filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			fmt.Println("failed to walk path", err)
			return err
		}
		command := strings.ReplaceAll(strings.TrimPrefix(path, root+"/"), "/", "-")
		if command == "" {
			command = "meme"
		}
		fin.Store(strings.TrimSuffix(command, filepath.Ext(command)), MemeHit{IsDir: info.IsDir(), Path: path})
		if info.IsDir() {
			if err := bm(filepath.Join(base, info.Name()), root, fin); err != nil {
				fmt.Println("failed to walk subdir", err)
				return err
			}
		}
		return nil
	})
}
