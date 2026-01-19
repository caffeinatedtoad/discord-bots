package pkg

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"log/slog"
	"marcus/pkg/tts"
	"marcus/pkg/util"
	"strings"
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
	msg := c.MessageEvent.Content

	if c.TTS == nil {
		c.TTS, _ = tts.NewTTS(c.Logger.With("component", "tts"))
	}

	voice, cmd, channel, content, isTTS, err := c.ExtractCommandParts(msg)
	if err != nil {
		c.err = err
		return c
	}

	c.Logger = c.Logger.With(
		"user", c.MessageEvent.Author.Username,
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

	// TODO: This needs to be rethought, since the
	//		 cache might be really big
	//if cmd == "list-cache" {
	//	c.action = c.SayCachedFiles
	//	c.usableOutsideOfVC = true
	//	return c
	//}

	if cmd == "" {
		c.ignore = true
		return c
	}

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
				memes := c.MemeSet.ListMemes()
				util.SendMessageWithError(c.Session, c.MessageEvent, memes, "failed list memes")
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

// parseChannelAndContent extracts an optional <channel> token and remaining content from the input string.
// If the input starts with <...>, it extracts the channel name and returns the rest as content.
// Otherwise, it returns the entire input as content.
func parseChannelAndContent(input string) (channel string, content string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", ""
	}

	if !strings.HasPrefix(input, "<") {
		return "", input
	}

	channelToken, remainder, found := strings.Cut(input, " ")
	if !found {
		// No space found, check if the entire input is a channel token
		if strings.HasPrefix(channelToken, "<") && strings.HasSuffix(channelToken, ">") {
			channel = strings.TrimPrefix(channelToken, "<")
			channel = strings.TrimSuffix(channel, ">")
			return channel, ""
		}
		return "", input
	}

	if strings.HasPrefix(channelToken, "<") && strings.HasSuffix(channelToken, ">") {
		channel = strings.TrimPrefix(channelToken, "<")
		channel = strings.TrimSuffix(channel, ">")
		content = strings.TrimSpace(remainder)
		return channel, content
	}

	return "", input
}

// splitCommandAndSubcommand splits a command string like "marcus" or "marcus-joke" into base and subcommand.
// Returns the base command and an optional subcommand.
func splitCommandAndSubcommand(commandString string) (base string, subcommand string) {
	before, after, found := strings.Cut(commandString, "-")
	if !found {
		return commandString, ""
	}
	return before, after
}

// buildCommand combines a base command with an optional subcommand.
func buildCommand(base string, subcommand string) string {
	if subcommand == "" {
		return base
	}
	return base + "-" + subcommand
}

// handleVoiceSyntax processes messages starting with "v!<voice>".
// It validates the voice, checks for conflicts with !marcus syntax, and extracts command parts.
func (c *Command) handleVoiceSyntax(msg string) (voice string, command string, channel string, content string, isTTS bool, err error) {
	// Handle special "v!voices" command - set command to "list-voices" and return isTTS=false
	if strings.HasPrefix(msg, "v!voices") {
		return "", "list-voices", "", "", false, nil
	}

	// Split "v!<voice>" from the rest of the message
	voiceToken, restOfMessage, hasRemainder := strings.Cut(msg, " ")

	// Extract voice name and optional subcommand from "v!<voice>[-<sub>]"
	voiceSpecifier := strings.TrimPrefix(voiceToken, "v!")
	voiceName, subcommand := splitCommandAndSubcommand(voiceSpecifier)

	// Validate that the voice exists
	_, validationError := c.GetGeneratorForVoice(voiceName)
	if validationError != nil {
		return "", "", "", "", false, fmt.Errorf("unknown voice '%s'", voiceName)
	}

	restOfMessage = strings.TrimSpace(restOfMessage)

	// If there's content after the voice token, check for conflicts
	if hasRemainder && restOfMessage != "" {
		firstToken, _, _ := strings.Cut(restOfMessage, " ")

		// Check if user is trying to combine v!<voice> with !marcus or !m
		commandCandidate := firstToken
		if strings.HasPrefix(commandCandidate, "!") {
			commandCandidate = strings.TrimPrefix(commandCandidate, "!")
		}

		// Extract base command name (before any - or < characters)
		baseCommand, _ := splitCommandAndSubcommand(commandCandidate)
		angleBracketIndex := strings.Index(baseCommand, "<")
		if angleBracketIndex != -1 {
			baseCommand = baseCommand[:angleBracketIndex]
		}

		// Reject combination of v!<voice> with !marcus or !m
		if baseCommand == "marcus" || baseCommand == "m" {
			return "", "", "", "", false, fmt.Errorf("do not combine v!<voice> with !marcus in the same message")
		}

		// If user explicitly provided another ! command, pass through voice only
		if strings.HasPrefix(restOfMessage, "!") {
			return voiceName, "", "", "", false, nil
		}
	}

	// Implicit marcus command when using v!<voice> syntax
	command = buildCommand("marcus", subcommand)

	// Parse optional channel and content from the rest of the message
	channel, content = parseChannelAndContent(restOfMessage)

	return voiceName, command, channel, content, true, nil
}

// handleMarcusSyntax processes messages starting with "!marcus" or "!m".
// It extracts the command, subcommand, channel, and content, using the default marcus voice.
func (c *Command) handleMarcusSyntax(msg string) (voice string, command string, channel string, content string, isTTS bool, err error) {
	// Split "!marcus" or "!m" from the rest of the message
	commandToken, restOfMessage, _ := strings.Cut(msg, " ")

	// Remove the "!" prefix to get "marcus" or "m[...]"
	commandName := strings.TrimPrefix(commandToken, "!")

	// Extract base and subcommand
	base, subcommand := splitCommandAndSubcommand(commandName)

	// Normalize "m" to "marcus"
	if base == "m" {
		base = "marcus"
	}

	// Build the full command
	command = buildCommand(base, subcommand)

	// Parse optional channel and content from the rest of the message
	restOfMessage = strings.TrimSpace(restOfMessage)
	channel, content = parseChannelAndContent(restOfMessage)

	// Use the default marcus voice
	voice = tts.MarcusDefaultVoice

	return voice, command, channel, content, true, nil
}

// ExtractCommandParts parses an incoming message and extracts voice, command, channel, and content.
// Supported forms:
// - "v!<voice> <content>"
// - "v!<voice>-<sub> [<channel>] [content]"
// - "!marcus[-<sub>] [<channel>] [content]"
// Returns isTTS=true for TTS commands (v!<voice> or !marcus), false for other commands like !ask-ai, !list-memes.
// It enforces that v!<voice> cannot be combined with !marcus/!m explicitly in the same message.
func (c *Command) ExtractCommandParts(msg string) (voice string, command string, channel string, content string, isTTS bool, err error) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", "", "", "", false, nil
	}

	// Handle v!<voice> syntax
	if strings.HasPrefix(msg, "v!") {
		return c.handleVoiceSyntax(msg)
	}

	// Handle !marcus or !m syntax
	if strings.HasPrefix(msg, "!marcus") || strings.HasPrefix(msg, "!m ") {
		return c.handleMarcusSyntax(msg)
	}

	if strings.HasPrefix(msg, "!") {
		return "", strings.TrimPrefix(msg, "!"), "", "", false, nil
	}

	// No recognized command syntax
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
