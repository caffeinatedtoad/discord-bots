package main

import (
	"fmt"
	"log"
	"log/slog"
	"marcus/pkg"
	"marcus/pkg/tts"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var logger *slog.Logger

func main() {

	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		slog.Error("please provide DISCORD_BOT_TOKEN in the environment variables")
		os.Exit(1)
	}

	if os.Getenv("magic_key") == "" {
		slog.Error("please provide Magic Key in the environment variables")
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		slog.Error(fmt.Sprintf("Error creating Discord session: %v", err))
		os.Exit(1)
	}

	marcus := NewMarcus()

	dg.AddHandler(marcus.handleMessage)
	err = dg.Open()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error opening connection: %v", err))
	}

	slog.Info("Bot is now running. Press CTRL-C to exit.")
	select {}
}

type Marcus struct {
	Memes *pkg.MemeSet
}

func NewMarcus() *Marcus {
	m := &Marcus{
		Memes: &pkg.MemeSet{
			Map: &sync.Map{},
		},
	}
	m.Memes.MonitorMemes(logger)
	return m
}

func (a *Marcus) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	ttsGen, err := tts.NewTTS(logger.With("component", "tts"))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create TTS generator: %v", err))
		return
	}

	c := pkg.Command{
		Session:      s,
		MessageEvent: m,
		Logger:       logger.With("ID", m.ID, "author", m.Author.Username, "channel", m.ChannelID),
		TTS:          ttsGen,
		MemeSet:      a.Memes,
	}

	err = c.Build().Execute()
	if err != nil {
		_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error executing command: %v", err))
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send error message: %v", err))
		}
	}
}
