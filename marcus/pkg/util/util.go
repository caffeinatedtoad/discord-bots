package util

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
)

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

func SendMessageWithError(e *events.MessageCreate, channelId snowflake.ID, content, errorMessage string) {
	_, err := SendMessageInChannel(e, channelId, content)
	if err != nil {
		_, _ = SendMessageInChannel(e, channelId, fmt.Sprintf("%s: %v", errorMessage, err))
		return
	}
}

func EditMessageWithError(e *events.MessageCreate, msgId snowflake.ID, content, errorMessage string) (*discord.Message, error) {
	msg, err := e.Client().Rest.UpdateMessage(e.ChannelID, msgId, discord.NewMessageUpdate().WithContent(content))
	if err != nil {
		_, _ = SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("%s: %v", errorMessage, err))
		return nil, err
	}
	return msg, err
}

func SendMessageInChannel(e *events.MessageCreate, channelId snowflake.ID, content string) (*discord.Message, error) {
	return e.Client().Rest.CreateMessage(channelId, discord.NewMessageCreate().WithContent(content))

}

func MessageInChannel(e *events.MessageCreate, channelName string) bool {
	cc, err := e.Client().Rest.GetGuildChannels(*e.GuildID)
	if err != nil {
		return true
	}

	for _, c := range cc {
		if strings.ToLower(c.Name()) == channelName && c.Type() == discord.ChannelTypeGuildText {
			return e.ChannelID == c.ID()
		}
	}
	return false
}

func GetUserVoiceChannel(e *events.MessageCreate, userID snowflake.ID) (*snowflake.ID, bool) {
	vs, err := e.Client().Rest.GetUserVoiceState(*e.GuildID, userID)
	if err != nil {
		return nil, false
	}
	return vs.ChannelID, true
}
