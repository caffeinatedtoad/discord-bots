package pkg

import (
	"fmt"
	"marcus/pkg/tts"
	"strings"
)

// extractVoiceCommand handles the v! and !marcus, or !m, syntax for requesting a voice command.
func extractVoiceCommand(msg string, generators []tts.Generator) (string, string, string, string, bool, error) {
	if !strings.HasPrefix(msg, "!") && !strings.HasPrefix(msg, "v!") {
		return "", "", "", "", false, nil
	}

	if strings.HasPrefix(msg, "v!voices") {
		return "", "list-voices", "", "", false, nil
	}

	trim := "!"
	if strings.HasPrefix(msg, "v!") {
		trim = "v!"
	}

	fullCommand, remainder, _ := strings.Cut(msg, " ")
	fullCommand = strings.TrimPrefix(fullCommand, trim)

	baseCommand, subcommand := splitCommandAndSubcommand(fullCommand)
	voice := ""
	// we know it's a TTS request based off of the leading v!
	if trim == "v!" {
		// Validate that the baseCommand exists
		if len(generators) == 0 {
			return "", "", "", "", false, fmt.Errorf("no voice generators configured")
		}
		supported := false
		for _, gen := range generators {
			// baseCommand is a voice (v!liam)
			if gen.SupportsVoice(strings.ToLower(baseCommand)) {
				supported = true
				break
			}
		}
		if !supported {
			return "", "", "", "", false, fmt.Errorf("unknown voice '%s'", baseCommand)
		}
		voice = baseCommand
	} else {
		voice = tts.DefaultVoice
	}

	restOfMessage := strings.TrimSpace(remainder)
	channel, content := parseChannelAndContent(restOfMessage)

	return voice, buildCommand(baseCommand, subcommand), channel, content, true, nil
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
