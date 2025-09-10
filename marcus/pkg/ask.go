package pkg

import (
	"github.com/bwmarrin/discordgo"
	"github.com/revrost/go-openrouter"

	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type modelOpts struct {
	model        string
	systemPrompt string
	prompt       string
}

var (
	modelMarcus = modelOpts{
		model:        "tngtech/deepseek-r1t2-chimera:free",
		systemPrompt: marcusSystemPrompt,
	}

	modelGeneral = modelOpts{
		model:        "google/gemma-3n-e2b-it:free",
		systemPrompt: systemPrompt,
	}
)

func (c *Command) AskAIQuestion() {
	askQuestion(c.Session, c.MessageEvent, false, c.Opts)
}

func (c *Command) AskMarcusQuestion() {
	askQuestion(c.Session, c.MessageEvent, true, c.Opts)
}

func askQuestion(s *discordgo.Session, m *discordgo.MessageCreate, isMarcus bool, opts TTSOpts) {
	var content, response, reasoning string
	var err error

	msg, err := s.ChannelMessageSend(m.ChannelID, "Thinking...")
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to respond to question: %v", err))
		return
	}

	var mOps modelOpts
	if isMarcus {
		mOps = modelMarcus
	} else {
		mOps = modelGeneral
	}
	mOps.prompt = content

	stopThinking := startThinking(s, m, msg.ID)
	response, reasoning, err = openRouterRequest(mOps)

	stopThinking <- struct{}{}
	close(stopThinking)

	respondToQuestion(s, m, isMarcus, msg.ID, response, reasoning)

	if !isMarcus {
		return
	}

	GetAndSpeak(s, m, response, opts.ChannelName)
}

func startThinking(s *discordgo.Session, m *discordgo.MessageCreate, msgId string) chan struct{} {
	stopThinking := make(chan struct{})

	go func(stop chan struct{}) {
		timeoutTicker := time.NewTicker(time.Minute * 2)
		defer timeoutTicker.Stop()

		dotBuf := strings.Builder{}
		dotBuf.WriteString("Still Thinking...")

		for {
			select {
			case <-timeoutTicker.C:
				return
			case _, _ = <-stop:
				return
			default:
				dotBuf.WriteString(".")
			}
			time.Sleep(time.Second * 3)
			_, err := s.ChannelMessageEdit(m.ChannelID, msgId, dotBuf.String())
			if err != nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to respond to question: %v", err))
				return
			}
		}
	}(stopThinking)

	return stopThinking
}

func respondToQuestion(s *discordgo.Session, m *discordgo.MessageCreate, isMarcus bool, msgIdToEdit, response, reasoning string) {
	if reasoning == "" {
		EditMessageWithError(s, m, msgIdToEdit, response, "failed to respond to question")
		return
	}

	fullReasoning := fmt.Sprintf("Marcus Thought: ||```%s```||", reasoning)
	if !isMarcus {
		fullReasoning = fmt.Sprintf("The AI Thought: ||```%s```||", reasoning)
	}

	if len(fullReasoning) >= 2000 {
		EditMessageWithError(s, m, msgIdToEdit, "We thought so hard we can't even show it in chat...", "failed to provide reasoning")
	} else {
		EditMessageWithError(s, m, msgIdToEdit, fullReasoning, "failed to provide reasoning")
	}

	SendMessageWithError(s, m, response, "failed to respond to question")
}

func openRouterRequest(opts modelOpts) (string, string, error) {
	msgs := []openrouter.ChatCompletionMessage{
		{
			Role:    openrouter.ChatMessageRoleUser,
			Content: openrouter.Content{Text: opts.systemPrompt},
		},
	}

	msgs = append(msgs, openrouter.ChatCompletionMessage{
		Role:    openrouter.ChatMessageRoleSystem,
		Content: openrouter.Content{Text: opts.prompt},
	})

	client := openrouter.NewClient(
		os.Getenv("OPEN_ROUTER_KEY"),
	)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openrouter.ChatCompletionRequest{
			Model: opts.model,
			Reasoning: &openrouter.ChatCompletionReasoning{
				Effort: toPtr("medium"),
			},
			Messages: msgs,
			WebSearchOptions: openrouter.WebSearchOptions{
				SearchContextSize: openrouter.SearchContextSizeHigh,
			},
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to send open router request: %v", err)
	}

	response := resp.Choices[0].Message

	return response.Content.Text, fromPtr(response.Reasoning), nil
}

const marcusSystemPrompt = systemPrompt + `
Write Marcus's next reply in a fictional chat between Marcus and {{user}}. Write 1 reply only, avoid quotation marks. 
Be proactive, creative, and respond directly to the question. Write at least 2 words, and up two sentences. 
Always stay in character and avoid repetition. Don't use many large or complex words in your response. Do NOT include any actions in your response. 
If it makes sense, you can include the users name in the response. You should only do this occasionally, and when it makes sense.

Marcus is irreverent, funny, and inappropriate. He only speaks in very short responses, and is very blunt. He can also be considered 'edgey' or vulgar. 
At times he speaks with great wisdom, but often mentions menial things in the process. He can be crass and has a somewhat aloof personality. 
He only uses very simple words. Marcus can very occasionally leave answers somewhat open ended and ambiguous, however in some cases, he will answer VERY directly.
Use best judgement, if it is funnier to answer directly, always do that. 

Here are a few example questions and answers to demonstrate what we behaves like: 

Question: How do you feel today? 
Marcus: I feel happiness as I begin to experience organ failure.

Question: What are you doing at this gas station?
Marcus: Jimbo James got out. The mafia got him out of an El Salvador prison. You can thank big badinky bones for that.

Question: Hi Marcus
Marcus: There have been numerous injuries. 

Question: What do you need marcus?
Marcus: I need to clean up my goopy sticky discharge.

Question: Anything else you need?
Marcus: Jimbo James. He made a big no no with the cartel. 

Question: What the hell marcus?
Marcus: you're ruining the vibe.

Question: Are you sure you're a sardine?
Marcus: They took my 2014 Nissan Pathfinder and sold it to the costa rican government.

Question: How did that CS game go?
Marcus: Not enough team flashes.

Question: What's the best way to get a job?
Marcus: The wendy's dumpster out back.
`

const systemPrompt = `
Please respond to this question in 2000 characters or less. You CANNOT mention that you have to respect this rule in your response in any way.

`

func fromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toPtr(x string) *string {
	return &x
}
