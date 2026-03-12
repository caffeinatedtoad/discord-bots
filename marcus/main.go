package main

import (
	"context"
	"fmt"
	"log/slog"
	"marcus/pkg"
	"marcus/pkg/tts"
	"os"
	"sync"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/godave/golibdave"
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

	m := NewMarcus()

	client, err := disgo.New(botToken,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuildMessages,
				gateway.IntentGuildVoiceStates,
				gateway.IntentMessageContent,
			),
		),
		bot.WithEventListenerFunc(m.handleMessage),
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(golibdave.NewSession),
		),
	)
	if err != nil {
		slog.Error("error while building disgo", slog.Any("err", err))
		return
	}

	defer client.Close(context.TODO())

	m.VoiceManager = client.VoiceManager

	if err = client.OpenGateway(context.TODO()); err != nil {
		slog.Error("errors while connecting to gateway", slog.Any("err", err))
		return
	}

	slog.Info("Bot is now running. Press CTRL-C to exit.")
	select {}
}

type Marcus struct {
	Memes        *pkg.MemeSet
	VoiceManager voice.Manager
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

func (a *Marcus) handleMessage(event *events.MessageCreate) {
	if event.Message.Author.Bot {
		return
	}

	ttsGen, err := tts.NewTTS(logger.With("component", "tts"), a.VoiceManager)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create TTS generator: %v", err))
		return
	}

	c := pkg.Command{
		MessageEvent: event,
		Logger:       logger.With("ID", event.Message.ID, "author", event.Message.Author.Username, "channel", event.ChannelID),
		TTS:          ttsGen,
		MemeSet:      a.Memes,
	}

	err = c.Build().Execute()
	if err != nil {
		_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageCreate().WithContent(fmt.Sprintf("Error executing command: %v", err)))
		if err != nil {
			logger.Error(fmt.Sprintf("failed to send error message: %v", err))
		}
	}
}
