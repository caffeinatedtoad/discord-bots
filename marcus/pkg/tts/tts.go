package tts

import (
	"bytes"
	"context"
	"io"
	"marcus/pkg/util"
	"math/rand"

	"github.com/caffeinatedtoad/dca"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"

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
	VoiceManager voice.Manager
	Logger       *slog.Logger
	Generators   []Generator
	Voice        string
}

type Generator interface {
	Name() string
	GenerateTTS(input, voice string) ([]byte, error)
	SupportsVoice(voice string) bool
	ListSupportedVoices() ([]string, error)
}

const (
	DefaultVoice = "marcus"
)

var (
	TooLongMessages = []string{
		"The requested TTS string was too long :( (must 300 characters or less)",
		"Your text is so long it made the TTS engine sweat profusely. (must 300 characters or less)",
		"The TTS engine just committed seppuku rather than process that wall of text. Congratulations. (must 300 characters or less)",
		"Your massive text dump made the TTS engine physically ill. Hope you're proud of yourself. (must 300 characters or less)",
	}
)

func NewTTS(logger *slog.Logger, manager voice.Manager) (*TTS, error) {
	if logger == nil {
		logger = slog.Default()
	}

	tts := &TTS{
		VoiceManager: manager,
		Logger:       logger,
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

func (t *TTS) CacheAudio(generatorName, provider, voice, content string, data []byte) {
	hash := generateHash(content)
	fileName := getCachePath(provider, voice, content)

	// don't use the data slice directly,
	// it's unsafe.
	var cacheData []byte
	copy(cacheData, data)

	// Ensure directory exists
	if err := ensureCacheDirectory(getProviderFromGeneratorName(generatorName), voice); err != nil {
		t.Logger.Error("failed to create cache directory", "err", err)
		return
	}

	// Write audio file
	err := os.WriteFile(fileName, cacheData, 0644)
	if err != nil {
		t.Logger.Error("failed writing ElevenLabs TTS file", "file", fileName, "err", err)
		return
	}

	t.Logger.Info("downloaded ElevenLabs TTS file", "file", fileName, "bytes", len(cacheData), "hash", hash)

	// Update metadata
	if err := updateMetadata(getProviderFromGeneratorName(getProviderFromGeneratorName(generatorName)), voice, hash, content, int64(len(cacheData)), t.Logger); err != nil {
		t.Logger.Warn("failed to update metadata", "err", err)
	}
}

func (t *TTS) InputIsCached(input string) (string, bool) {
	voice := strings.TrimSpace(t.Voice)
	if voice == "" {
		voice = DefaultVoice
	}

	// Try to determine provider from the voice
	generator, err := t.GetGeneratorForVoice(voice)
	if err != nil {
		// Fallback to legacy check if no generator found
		fileName := getFileName(input, voice)
		return fileName, fileIsCached(fileName)
	}

	provider := getProviderFromGeneratorName(generator.Name())
	fileName := getFileNameWithFallback(provider, voice, input, t.Logger)
	return fileName, fileIsCached(fileName)
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

func (t *TTS) GenerateAndPlay(e *events.MessageCreate, content, targetChannel string) {
	if len(content) >= 1000 {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, TooLongMessages[rand.Intn(len(TooLongMessages))])
		return
	}

	var channelID *snowflake.ID
	var err error
	if targetChannel != "" {
		channelID, err = t.getVoiceChannelByName(e, targetChannel)
		if err != nil {
			_, err = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannel, err))
			t.Logger.Error("voice channel lookup failed", "targetChannel", targetChannel, "err", err)
			return
		}
	}

	voice := strings.ToLower(strings.TrimSpace(t.Voice))
	if voice == "" {
		voice = DefaultVoice
	}

	// Get generator for the voice
	generator, err := t.GetGeneratorForVoice(voice)
	if err != nil {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to find TTS generator for voice '%s': %v", voice, err))
		t.Logger.Error("failed to find TTS generator", "voice", voice, "err", err)
		return
	}

	// Determine provider and check cache with fallback
	provider := getProviderFromGeneratorName(generator.Name())
	fileName := getFileNameWithFallback(provider, voice, content, t.Logger)
	var audio []byte

	if !fileIsCached(fileName) {
		t.Logger.Info("TTS not cached, generating", "file", fileName, "voice", voice, "provider", provider)
		audio, err = generator.GenerateTTS(content, voice)
		if err != nil {
			_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to generate TTS: %v\n Type !marcus-cache to see all cached files that can be played at any time.", err))
			t.Logger.Error("failed to generate TTS", "file", fileName, "err", err)
			return
		}

		t.CacheAudio(generator.Name(), provider, voice, content, audio)

	} else {
		t.Logger.Info("using cached TTS", "file", fileName, "voice", voice, "provider", provider)
		audio, err = os.ReadFile(fileName)
		if err != nil {
			_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to read cached TTS file: %v", err))
			t.Logger.Error("failed to read cached TTS file", "file", fileName, "err", err)
			return
		}
	}

	if channelID != nil {
		go t.speakInChannel(e, audio, channelID)
	} else {
		go t.speakInUserVoiceChannel(e, audio)
	}
}

func (t *TTS) SpeakFile(e *events.MessageCreate, file string, targetChannelName string) {
	if !fileIsCached(file) {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("file '%s' not found in cache", file))
		return
	}

	audio, err := os.ReadFile(file)
	if err != nil {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to read cached TTS file: %v", err))
		return
	}

	if targetChannelName == "" {
		go t.speakInUserVoiceChannel(e, audio)
	}

	channelID, err := t.getVoiceChannelByName(e, targetChannelName)
	if err != nil {
		_, err = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannelName, err))
		return
	}

	go t.speakInChannel(e, audio, channelID)
}

func (t *TTS) Speak(e *events.MessageCreate, audio []byte, targetChannelName string) {
	if targetChannelName == "" {
		go t.speakInUserVoiceChannel(e, audio)
	}

	channelID, err := t.getVoiceChannelByName(e, targetChannelName)
	if err != nil {
		_, err = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to find voice channel with name '%s': %v", targetChannelName, err))
		return
	}

	go t.speakInChannel(e, audio, channelID)
}

func (t *TTS) speakInUserVoiceChannel(e *events.MessageCreate, audio []byte) {
	targetVoiceChannelId, foundInVC := util.GetUserVoiceChannel(e, e.Message.Author.ID)
	if !foundInVC {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, "you need to be in a voice channel to use this command")
		return
	}

	t.speakInChannel(e, audio, targetVoiceChannelId)
}

func (t *TTS) speakInChannel(e *events.MessageCreate, audio []byte, targetVoiceChannelId *snowflake.ID) {
	t.Logger.Info("joining voice channel to play audio", "guild", e.GuildID, "channelID", targetVoiceChannelId)

	encodeSession, err := dca.EncodeMem(bytes.NewReader(audio), dca.StdEncodeOptions)
	if err != nil {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to create encoding session: %v", err))
		return
	}
	defer encodeSession.Cleanup()

	t.Logger.Info("Starting to play audio")
	voiceConn := t.VoiceManager.CreateConn(*e.GuildID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = voiceConn.Open(ctx, *targetVoiceChannelId, false, true)
	if err != nil {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to join voice channel: %v", err))
		return
	}
	defer voiceConn.Close(ctx)

	err = voiceConn.SetSpeaking(ctx, voice.SpeakingFlagMicrophone)
	if err != nil {
		_, _ = util.SendMessageInChannel(e, e.ChannelID, fmt.Sprintf("failed to start speaking: %v", err))
		return
	}

	// drop any incoming audio, this is expected by the discord API.
	// we auto deafen, so we may not even get anything, but it can't hurt
	// to do what's expected here.
	go func() {
	FakeRead:
		for {
			select {
			case <-ctx.Done():
				break FakeRead
			default:
				// drop any errors,
				_, err = voiceConn.UDP().ReadPacket()
				if err != nil {
					break FakeRead
				}
				time.Sleep(time.Millisecond * 20)
			}
		}
	}()

	for {
		frame, err := encodeSession.OpusFrame()
		if err != nil {
			if err == io.EOF {
				t.Logger.Info("finished playing audio")
				break
			}
			t.Logger.Error("failed to read opus frame", "err", err)
			break
		}

		_, err = voiceConn.UDP().Write(frame)
		if err != nil {
			t.Logger.Error("failed to write packet", "err", err)
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Logger.Info("disconnected from voice channel", "channelID", targetVoiceChannelId)
	time.Sleep(time.Millisecond * 200)
}

func (t *TTS) getVoiceChannelByName(e *events.MessageCreate, channelName string) (*snowflake.ID, error) {
	cc, err := e.Client().Rest.GetGuildChannels(*e.GuildID)
	if err != nil {
		return nil, err
	}

	for _, c := range cc {
		if c.Name() == channelName {
			return new(c.ID()), nil
		}
	}

	return nil, fmt.Errorf("failed to find channel with the name %s", channelName)
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
