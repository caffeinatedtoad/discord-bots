package tts

import (
	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"marcus/pkg/util"

	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Opts struct {
	ChannelName string
	Filepath    string
	Content     string
}

type TTS struct {
	Logger     *slog.Logger
	Generators []Generator
	Voice      string // provider-specific voice id (e.g., TikTok text_speaker)
}

type Generator interface {
	Name() string
	GenerateTTS(input, voice string) (string, error)
	SupportsVoice(voice string) bool
	ListSupportedVoices() ([]string, error)
}

const (
	DefaultVoice = "marcus"

	// MarcusDefaultVoice is the default voice when using the !marcus family of commands
	// It may differ from DefaultVoice; keep equal by default and adjust as desired.
	MarcusDefaultVoice = DefaultVoice
)

var (
	TTSTooLongMessages = []string{
		"The requested TTS string was too long :( (must 300 characters or less)",
		"Your text is so long it made the TTS engine sweat profusely. (must 300 characters or less)",
		"The TTS engine just committed seppuku rather than process that wall of text. Congratulations. (must 300 characters or less)",
		"Your massive text dump made the TTS engine physically ill. Hope you're proud of yourself. (must 300 characters or less)",
	}
)

func NewTTS(logger *slog.Logger) (*TTS, error) {
	if logger == nil {
		logger = slog.Default()
	}

	tts := &TTS{
		Logger: logger,
		Generators: []Generator{
			// TODO: Use tiktok Api again at some point
			//TikTokTTSGenerator{
			//	HTTPClient: http.DefaultClient,
			//	Logger:     logger,
			//},
			CacheTTSGenerator{
				Logger: logger,
			},
		},
		Voice: DefaultVoice,
	}

	elevenlabs, err := NewElevenLabsTTSGenerator(logger)
	if err != nil {
		logger.Error("failed to initialize elevenlabs TTS generator", "err", err)
	} else {
		tts.Generators = append(tts.Generators, elevenlabs)
	}

	return tts, nil
}

func (t *TTS) ListSupportedVoiceNames() []string {
	var names []string
	for _, gen := range t.Generators {
		voices, err := gen.ListSupportedVoices()
		if err != nil {
			t.Logger.Error("failed to list supported voices", "generator", gen.Name(), "err", err)
			continue
		}
		names = append(names, voices...)
	}
	sort.Strings(names)
	return names
}

func (t *TTS) GetGeneratorForVoice(voice string) (Generator, error) {
	for _, gen := range t.Generators {
		if gen.SupportsVoice(voice) {
			return gen, nil
		}
	}
	return nil, fmt.Errorf("no generator found for voice: %s", voice)
}

func (t *TTS) GenerateAndPlay(s *discordgo.Session, m *discordgo.MessageCreate, content, targetChannel string) {
	//if len(content) >= 300 {
	//	s.ChannelMessageSend(m.ChannelID, TTSTooLongMessages[rand.Intn(len(TTSTooLongMessages))])
	//	return
	//}

	var channelID string
	var err error
	if targetChannel != "" {
		channelID, err = t.getVoiceChannelByName(s, m, targetChannel)
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannel, err))
			t.Logger.Error("voice channel lookup failed", "targetChannel", targetChannel, "err", err)
			return
		}
	}

	voice := strings.TrimSpace(t.Voice)
	if voice == "" {
		voice = DefaultVoice
	}

	fileName := getFileName(content, voice)
	if !fileIsCached(fileName) {
		t.Logger.Info("TTS not cached, generating", "file", fileName, "voice", voice)
		generator, err := t.GetGeneratorForVoice(voice)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find TTS generator for voice '%s': %v", voice, err))
			t.Logger.Error("failed to find TTS generator", "voice", voice, "err", err)
			return
		}

		fileName, err = generator.GenerateTTS(content, voice)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to generate TTS: %v\n Type !marcus-cache to see all cached files that can be played at any time.", err))
			t.Logger.Error("failed to generate TTS", "file", fileName, "err", err)
			return
		}
	} else {
		t.Logger.Info("using cached TTS", "file", fileName, "voice", voice)
	}

	if channelID != "" {
		t.speakFileInChannel(s, m, fileName, channelID)
	} else {
		t.speakFileInUserVoiceChannel(s, m, fileName)
	}
}

func (t *TTS) SpeakFile(s *discordgo.Session, m *discordgo.MessageCreate, fileName, targetChannelName string) {
	_, err := os.Stat(fileName)
	if err != nil {
		_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find file: %v", err))
		return
	}

	if targetChannelName != "" {
		channelID, err := t.getVoiceChannelByName(s, m, targetChannelName)
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannelName, err))
			return
		}
		t.speakFileInChannel(s, m, fileName, channelID)
		return
	}
	t.speakFileInUserVoiceChannel(s, m, fileName)
}

func (t *TTS) speakFileInChannel(s *discordgo.Session, m *discordgo.MessageCreate, fileName, targetVoiceChannelId string) {
	t.Logger.Info("joining voice channel to play audio", "guild", m.GuildID, "channelID", targetVoiceChannelId, "file", fileName)
	vc, err := s.ChannelVoiceJoin(m.GuildID, targetVoiceChannelId, false, true)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to connect to the vc: %v", err))
		t.Logger.Error("voice join failed", "channelID", targetVoiceChannelId, "err", err)
		return
	}

	time.Sleep(time.Millisecond * 250)
	dgvoice.PlayAudioFile(vc, fileName, make(chan bool))
	time.Sleep(time.Second * 1)

	err = vc.Disconnect()
	if err != nil {
		t.Logger.Error("failed to disconnect from voice channel", "err", err)
		return
	}
	t.Logger.Info("disconnected from voice channel", "channelID", targetVoiceChannelId)
}

func (t *TTS) speakFileInUserVoiceChannel(s *discordgo.Session, m *discordgo.MessageCreate, file string) {
	targetVoiceChannelId, foundInVC := util.GetUserVoiceChannel(s, m.Author.ID)
	if !foundInVC {
		_, _ = s.ChannelMessageSend(m.ChannelID, "you need to be in a voice channel to use this command")
		return
	}
	t.speakFileInChannel(s, m, file, targetVoiceChannelId)
}

func (t *TTS) getVoiceChannelByName(s *discordgo.Session, m *discordgo.MessageCreate, channelName string) (string, error) {
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

func getFileName(input, voice string) string {
	audioDir := os.Getenv("AUDIO_DIR")
	if audioDir == "" {
		audioDir = filepath.Join(".", "audio")
	}
	// Ensure directory exists (best effort)
	_ = os.MkdirAll(audioDir, 0755)

	// Build base name with spaces preserved but sanitize invalid filesystem chars
	base := strings.TrimSpace(input)
	base = strings.TrimSuffix(base, ".")
	base = sanitizeFileName(base)
	// Replace remaining runs of whitespace with single underscore for consistency
	ws := regexp.MustCompile(`\s+`)
	base = ws.ReplaceAllString(base, "_")
	base = strings.Trim(base, " ._")
	if base == "" {
		base = "tts"
	}
	voiceSafe := strings.TrimSpace(voice)
	if voiceSafe == "" {
		voiceSafe = DefaultVoice
	}
	voiceSafe = sanitizeFileName(voiceSafe)
	full := filepath.Join(audioDir, voiceSafe+"__"+base) + ".wav"

	// Backward compatibility: if legacy name exists, use it
	legacy := legacyFileName(input)
	if fileIsCached(legacy) {
		return legacy
	}
	return full
}

// legacyFileName replicates the old naming (spaces as underscores, no sanitization)
func legacyFileName(input string) string {
	name := strings.Join(strings.Split(input, " "), "_")
	name = strings.TrimSuffix(name, ".")
	audioDir := os.Getenv("AUDIO_DIR")
	if audioDir == "" {
		audioDir = filepath.Join(".", "audio")
	}
	return filepath.Join(audioDir, fmt.Sprintf("%s.wav", name))
}

// sanitizeFileName replaces characters invalid on Windows and trims unsafe edges
func sanitizeFileName(name string) string {
	// Replace invalid characters: <>:"/\\|?* and control chars
	invalid := regexp.MustCompile(`[<>:\"/\\|?*\x00-\x1F]`)
	name = invalid.ReplaceAllString(name, "_")
	// Collapse multiple underscores
	multiUnderscore := regexp.MustCompile(`_+`)
	name = multiUnderscore.ReplaceAllString(name, "_")
	return name
}

func fileIsCached(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
