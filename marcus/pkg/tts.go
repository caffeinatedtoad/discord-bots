package pkg

import (
	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"time"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TTSOpts struct {
	ChannelName string
	Filepath    string
	Content     string
}

type TargetChannel string

const (
	TargetCurrentUserChannel = ""
)

func (c *Command) SayTTS() {
	GetAndSpeak(c.Session, c.MessageEvent, c.Opts.Content, c.Opts.ChannelName)
}

func GetAndSpeak(s *discordgo.Session, m *discordgo.MessageCreate, content, targetChannel string) {
	if len(content) >= 300 {
		s.ChannelMessageSend(m.ChannelID, "The requested TTS string was too long :(")
		return
	}

	channelID := TargetCurrentUserChannel
	var err error
	if targetChannel != TargetCurrentUserChannel {
		channelID, err = getVoiceChannelByName(s, m, targetChannel)
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannel, err))
			return
		}
	}

	fileName := getFileName(content)
	if !fileIsCached(fileName) {
		log.Println("phrase not cached, generating TTS: ", fileName)
		fileName, err = getTTS(content)
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to generate TTS: %v\n Type !marcus-cache to see all cached files that can be played at any time.", err))
			return
		}
	} else {
		log.Println("using cached file for: ", fileName)
	}

	if channelID != "" {
		speakFileInChannel(s, m, fileName, channelID)
	} else {
		speakFileInUserVoiceChannel(s, m, fileName)
	}
}

func SpeakFile(s *discordgo.Session, m *discordgo.MessageCreate, fileName, targetChannelName string) {
	_, err := os.Stat(fileName)
	if err != nil {
		_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find file: %v", err))
		return
	}

	if targetChannelName != "" {
		channelID, err := getVoiceChannelByName(s, m, targetChannelName)
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannelName, err))
			return
		}
		speakFileInChannel(s, m, fileName, channelID)
	} else {
		speakFileInUserVoiceChannel(s, m, fileName)
	}
}

func speakFileInChannel(s *discordgo.Session, m *discordgo.MessageCreate, fileName, targetVoiceChannelId string) {
	vc, err := s.ChannelVoiceJoin(m.GuildID, targetVoiceChannelId, false, true)
	if err != nil {
		_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to connect to the vc: %v", err))
		return
	}

	dgvoice.PlayAudioFile(vc, fileName, make(chan bool))
	time.Sleep(time.Millisecond * 500)
	err = vc.Disconnect()
	if err != nil {
		log.Println(fmt.Sprintf("Failed to disconnect from the vc: %v", err))
		return
	}
}

func speakFileInUserVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate, file string) {
	targetVoiceChannelId, foundInVC := GetUserVoiceChannel(s, m.Author.ID)
	if !foundInVC {
		_, _ = s.ChannelMessageSend(m.ChannelID, "you need to be in a voice channel to use this command")
		return
	}
	speakFileInChannel(s, m, file, targetVoiceChannelId)
}

func getVoiceChannelByName(s *discordgo.Session, m *discordgo.MessageCreate, channelName string) (string, error) {
	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return "", err
	}

	cc, err := s.GuildChannels(guild.ID)
	if err != nil {
		return "", err
	}

	for _, c := range cc {
		if c.Name == channelName && c.Type == discordgo.ChannelTypeGuildVoice {
			return c.ID, nil
		}
	}

	return "", fmt.Errorf("failed to find channel with the name %s", channelName)
}

func getFileName(input string) string {
	name := strings.Join(strings.Split(input, " "), "_")
	name = strings.TrimSuffix(name, ".")

	name = filepath.Join(os.Getenv("AUDIO_DIR"), name)

	return fmt.Sprintf("%s.wav", name)
}

func fileIsCached(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func getTTS(input string) (string, error) {
	req, _ := http.NewRequest(http.MethodPost, "https://api16-normal-useast5.us.tiktokv.com/media/api/text/speech/invoke/", nil)
	req.Header.Set("User-Agent", "com.zhiliaoapp.musically/2022600030 (Linux; U; Android 7.1.2; es_ES; SM-G988N; Build/NRD90M;tt-ok/3.12.13.1)")
	req.Header.Set("Cookie", fmt.Sprintf("sessionid=%s", os.Getenv("magic_key")))

	q := req.URL.Query()
	q.Set("aid", "1233")
	q.Set("speaker_map_type", "0")
	q.Set("text_speaker", "en_male_narration")
	q.Set("req_text", input)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", nil
	}
	resp.Body.Close()

	tt := ttsResponse{}
	err = json.Unmarshal(b, &tt)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(tt.Data.VStr)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	os.WriteFile(getFileName(input), data, 0644)
	log.Println("downloaded tts")
	return getFileName(input), nil
}

type ttsResponse struct {
	Data struct {
		SKey     string `json:"s_key"`
		VStr     string `json:"v_str"`
		Duration string `json:"duration"`
		Speaker  string `json:"speaker"`
	} `json:"data"`
	Extra struct {
		LogID string `json:"log_id"`
	} `json:"extra"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}
