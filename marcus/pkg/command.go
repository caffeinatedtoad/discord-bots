package pkg

import (
	"github.com/bwmarrin/discordgo"

	"fmt"
	"log/slog"
	"strings"
)

type Command struct {
	Logger *slog.Logger

	Opts         TTSOpts
	Session      *discordgo.Session
	MessageEvent *discordgo.MessageCreate

	err               error
	ignore            bool
	action            func()
	usableOutsideOfVC bool
}

func (c *Command) Build() *Command {
	msg := c.MessageEvent.Content
	command, content, _ := strings.Cut(msg, " ")
	isCommand := strings.HasPrefix(command, "!")
	if !isCommand {
		c.ignore = true
		return c
	}

	command = strings.TrimPrefix(command, "!")

	targetChannel := ""
	if strings.Contains(command, "<") {
		commandSplit := strings.Split(command, "<")
		command = commandSplit[0]
		targetChannel = strings.TrimSuffix(commandSplit[1], ">")
		content = strings.TrimSpace(content)
	}

	if targetChannel != "" {
		c.usableOutsideOfVC = true
	}

	c.Logger = c.Logger.With(
		"user", c.MessageEvent.Author.Username,
		"guildID", c.MessageEvent.GuildID,
		"command", command,
		"content", content,
		"targetChannel", targetChannel,
	)

	defer func() {
		if c.ignore {
			return
		}

		if c.err != nil {
			c.Logger.Error(fmt.Sprintf("failed to build command: %v", c.err))
		} else {
			c.Logger.Info("Built command")
		}
	}()

	c.Opts = TTSOpts{
		Content:     content,
		ChannelName: targetChannel,
	}

	command, subcommand, hasSubCommand := strings.Cut(command, "-")

	if strings.HasPrefix(command, "marcus") || command == "m" {
		if !hasSubCommand {
			c.action = c.SayTTS
			return c
		}
		switch subcommand {
		case "cache":
			c.action = c.SayCachedFiles
			c.usableOutsideOfVC = true
		case "insult":
			c.action = c.SayInsult
		case "fact":
			c.action = c.SayFact
		case "joke":
			c.action = c.SayJoke
		case "slur":
			c.action = c.SaySlur
		default:
			c.ignore = true
		}
		return c
	}

	if strings.HasPrefix(command, "ask") {
		switch subcommand {
		case "marcus":
			c.action = c.AskMarcusQuestion
		case "ai":
			c.action = c.AskAIQuestion
			c.usableOutsideOfVC = true
		default:
			c.err = fmt.Errorf("unknown !ask subcommand: %s", subcommand)
			return c
		}
		return c
	}

	if command == "list" {
		switch subcommand {
		case "memes":
			c.action = c.ListMemes
			c.usableOutsideOfVC = true
			return c
		}
	}

	meme, ok := Memes.Memes[command]
	if ok {
		c.Meme(meme, subcommand)
	}

	c.ignore = true
	return c
}

func (c *Command) Execute() error {
	if c.ignore || c.err != nil {
		return nil
	}

	funcName := GetFunctionName(c.action)
	if funcName == "unknown" {
		// should never happen
		return fmt.Errorf("failed to get function name for command")
	}

	c.Logger = c.Logger.With("func", funcName)

	_, userInVC := GetUserVoiceChannel(c.Session, c.MessageEvent.Author.ID)
	if !c.usableOutsideOfVC && !userInVC {
		_, err := c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, "You must be in a voice channel to use this command.")
		if err != nil {
			c.Logger.Error(fmt.Sprintf("encountered error sending message: %v", err))
		}
		return nil
	}

	c.Logger.Info("Executing command")
	c.action()

	return nil
}
