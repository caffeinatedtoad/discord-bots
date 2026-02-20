package pkg

import (
	"fmt"
	"log"
	"log/slog"
	"marcus/pkg/tts"
	"marcus/pkg/util"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Logger *slog.Logger

	*tts.TTS
	*MemeSet

	TTSOpts tts.Opts

	Session      *discordgo.Session
	MessageEvent *discordgo.MessageCreate

	err               error
	ignore            bool
	action            func()
	usableOutsideOfVC bool

	CommandString    string
	SubcommandString string
}

func (c *Command) Build() *Command {
	voice, cmd, channel, content, isTTS, err := c.ExtractCommandParts(c.MessageEvent.Content)
	if err != nil {
		c.Logger.Error(fmt.Sprintf("failed to extract command parts: %v", err))
		c.err = err
		return c
	}

	c.Logger.Debug("Extracted command parts", "voice", voice, "cmd", cmd, "channel", channel, "content", content, "isTTS", isTTS)

	if cmd == "" {
		c.ignore = true
		return c
	}

	if c.TTS == nil {
		c.TTS, _ = tts.NewTTS(c.Logger.With("component", "tts"))
	}

	c.Logger = c.Logger.With(
		"guildID", c.MessageEvent.GuildID,
		"command", cmd,
		"content", content,
		"targetChannel", channel,
	)

	if cmd == "list-voices" {
		c.action = c.ListVoices
		c.usableOutsideOfVC = true
		return c
	}

	// TODO: this will be dedicated to the UI only.
	//if cmd == "list-cache" {
	//	c.action = c.SayCachedFiles
	//	c.usableOutsideOfVC = true
	//	return c
	//}

	// Set voice if provided (v! path) or default will be used later for !marcus path
	if voice != "" {
		c.TTS.Voice = voice
	}

	c.TTSOpts = tts.Opts{Content: content, ChannelName: channel}
	base := cmd
	sub := ""
	if idx := strings.Index(cmd, "-"); idx != -1 {
		base = cmd[:idx]
		sub = cmd[idx+1:]
	}
	c.CommandString = base
	c.SubcommandString = sub

	if channel != "" {
		c.usableOutsideOfVC = true
	}

	if isTTS {
		switch sub {
		case "insult":
			c.action = c.SayInsult
			return c
		case "fact":
			c.action = c.SayFact
			return c
		case "joke":
			c.action = c.SayJoke
			return c

		// allow saying cached slurs,
		// but don't generate any more
		case "slur":
			c.action = c.SaySlur
			return c

		default:
			// Unknown subcommand, treat as plain TTS if content provided
			if strings.TrimSpace(content) == "" {
				c.action = func() {
					util.SendMessageWithError(c.Session, c.MessageEvent, "Unknown subcommand. Try 'v!voices' or provide a message.", "invalid subcommand")
				}
				c.usableOutsideOfVC = true
				return c
			}
			c.action = func() {
				c.TTS.GenerateAndPlay(c.Session, c.MessageEvent, content, channel)
			}
			return c
		}
	}

	if strings.HasPrefix(c.CommandString, "ask") {
		switch c.SubcommandString {
		case "marcus":
			c.action = c.AskMarcusQuestion
		case "ai":
			c.action = c.AskAIQuestion
			c.usableOutsideOfVC = true
		default:
			c.err = fmt.Errorf("unknown !ask subcommand: %s", c.SubcommandString)
			return c
		}
		return c
	}

	if c.CommandString == "list" {
		switch c.SubcommandString {
		case "memes":
			c.action = func() {
				util.SendMessageWithError(c.Session, c.MessageEvent, c.MemeSet.ListMemes(), "failed list memes")
			}
			c.usableOutsideOfVC = true
			return c
		}
	}

	if c.CommandString == "addmeme" {
		c.action = c.AddMeme
		c.usableOutsideOfVC = true
		return c
	}

	meme, found := c.MemeSet.GetMeme(cmd)
	if found {
		c.Logger.Info("found meme for ", cmd, "")
		c.action = func() {
			c.TTS.SpeakFile(c.Session, c.MessageEvent, meme, channel)
		}
		return c
	} else {
		c.Logger.Info("didn't find a meme for ", cmd)
	}

	c.ignore = true
	return c
}

func (c *Command) Execute() error {
	if c.ignore || c.err != nil {
		return nil
	}

	funcName := util.GetFunctionName(c.action)
	if funcName == "unknown" {
		// should never happen
		return fmt.Errorf("failed to get function name for command")
	}

	c.Logger = c.Logger.With("func", funcName)

	_, userInVC := util.GetUserVoiceChannel(c.Session, c.MessageEvent.Author.ID)
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

// ExtractCommandParts parses an incoming message and extracts voice, command, channel, and content.
// Supported forms:
// - "v!<voice> <content>"
// - "v!<voice>-<sub> [<channel>] [content]"
// - "!marcus[-<sub>] [<channel>] [content]"
// Returns isTTS=true for TTS commands (v!<voice> or !marcus), false for other commands like !ask-ai, !list-memes.
// It enforces that v!<voice> cannot be combined with !marcus/!m explicitly in the same message.
func (c *Command) ExtractCommandParts(msg string) (string, string, string, string, bool, error) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", "", "", "", false, nil
	}

	// Handle v!<voice> syntax
	if strings.HasPrefix(msg, "v!") || strings.HasPrefix(msg, "!marcus") || strings.HasPrefix(msg, "!m ") {
		return extractVoiceCommand(msg, c.Generators)
	}

	if strings.HasPrefix(msg, "!") {
		return "", strings.TrimPrefix(msg, "!"), "", "", false, nil
	}

	// No recognized command syntax, just a normal message
	return "", "", "", "", false, nil
}

func (c *Command) ListVoices() {
	names := make(map[string][]string)
	for _, gen := range c.Generators {
		voices, err := gen.ListSupportedVoices()
		if err != nil {
			c.Logger.Error(fmt.Sprintf("failed to list voices for %s: %v", gen.Name(), err))
			continue
		}
		names[gen.Name()] = voices
	}
	if len(names) == 0 {
		util.SendMessageWithError(c.Session, c.MessageEvent, "No voices are currently configured.", "list voices")
		return
	}

	b := strings.Builder{}
	b.WriteString("Supported voices:\n")
	for k, v := range names {
		gen := fmt.Sprintf("Platform: %s\n", k)
		for _, e := range v {
			gen = gen + fmt.Sprintf("	- Voice: %s\n", e)
		}
		b.WriteString(gen)
	}

	b.WriteString("\na few examples: \nv!liam [laughing] I know jor jor well!\nv!alice [energetic] it's all about the mets baby!\nv!sarah [questioning] surely one more game won't be incredibly tilting, right?\nv!liam-insult\nv!alice-joke\n")

	util.SendMessageWithError(c.Session, c.MessageEvent, fmt.Sprintf("```\n%s\n```", b.String()), "failed to list voices")
}

func (c *Command) SayCachedFiles() {
	cached := util.GetCachedFiles()

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
