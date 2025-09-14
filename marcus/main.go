package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"log/slog"
	"marcus/pkg"
	"os"
)

var logger *slog.Logger

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

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

	dg.AddHandler(handleMessage)
	err = dg.Open()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error opening connection: %v", err))
	}

	slog.Info("Bot is now running. Press CTRL-C to exit.")
	select {}
}

func handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	c := pkg.Command{
		Session:      s,
		MessageEvent: m,
		Logger:       logger,
	}

	err := c.Build().Execute()

	if err != nil {
		_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error executing command: %v", err))
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send error message: %v", err))
		}
	}
}
