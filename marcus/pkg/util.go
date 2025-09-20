package pkg

import (
	"github.com/bwmarrin/discordgo"

	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

func (c *Command) SayCachedFiles() {
	cached := getCachedFiles()

	b := strings.Builder{}
	var parts []string
	b.WriteString("```\n")
	for _, file := range cached {
		b.WriteString(fmt.Sprintf("%s\n", file))
		if len(b.String()) >= 200 {
			// write b and continue to build the next page
			b.WriteString("```\n")
			parts = append(parts, b.String())
			b.Reset()
			b.WriteString("```\n")
		}
	}
	b.WriteString("```\n")

	parts = append(parts, b.String())
	if len(parts) == 1 {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, b.String())
		if err != nil {
			log.Println(fmt.Sprintf("ERR: %v", err))
		}
		return
	}

	for i, part := range parts {
		part = fmt.Sprintf("Page %d\n%s", i, part)
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, part)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: %v", err))
			return
		}
	}
}

func getCachedFiles() []string {
	var phrases []string
	dir := "."
	if os.Getenv("AUDIO_DIR") != "" {
		dir = os.Getenv("AUDIO_DIR")
	}

	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		info, err := d.Info()
		if err != nil {
			return nil
		}

		if strings.Contains(info.Name(), ".wav") {
			phrase := strings.TrimSuffix(info.Name(), ".wav")
			phrase = strings.ReplaceAll(phrase, "_", " ")
			phrase = strings.TrimPrefix(phrase, " ")
			phrases = append(phrases, phrase)
		}
		return nil
	})

	return phrases
}

func GetRandomEmbedTitle() string {
	options := []string{
		"Thoughts are now crimes. Welcome to the future",
		"1984 wasn’t a warning. It was a blueprint",
		"This is what happens when freedom is taken away one word at a time",
		"Is ‘thinking’ a crime now, or is that just the next step?",
	}
	return options[rand.Intn(len(options))]
}

func GetFunctionName(i interface{}) string {
	// Get the function value using reflection
	funcValue := reflect.ValueOf(i)

	// Get the pointer to the function
	funcPointer := runtime.FuncForPC(funcValue.Pointer())
	if funcPointer == nil {
		return "unknown"
	}

	// Return the name of the function
	return funcPointer.Name()
}

// TODO: This is really just handling the case where the message length is too long - need a new function to do proper logging

func SendMessageWithError(s *discordgo.Session, m *discordgo.MessageCreate, content, errorMessage string) {
	_, err := s.ChannelMessageSend(m.ChannelID, content)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s: %v", errorMessage, err))
		return
	}
}

func EditMessageWithError(s *discordgo.Session, m *discordgo.MessageCreate, msgId, content, errorMessage string) {
	_, err := s.ChannelMessageEdit(m.ChannelID, msgId, content)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s: %v", errorMessage, err))
		return
	}
}

func messageInChannel(s *discordgo.Session, m *discordgo.MessageCreate, channelName string) bool {
	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return true
	}

	cc, err := s.GuildChannels(guild.ID)
	if err != nil {
		return true
	}

	for _, c := range cc {
		if strings.ToLower(c.Name) == channelName && c.Type == discordgo.ChannelTypeGuildText {
			return m.ChannelID == c.ID
		}
	}

	return false
}

func GetUserVoiceChannel(s *discordgo.Session, userID string) (string, bool) {
	for _, guild := range s.State.Guilds {
		for _, vs := range guild.VoiceStates {
			if vs.UserID == userID {
				return vs.ChannelID, true
			}
		}
	}
	return "", false
}
