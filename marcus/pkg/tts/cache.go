package tts

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type CacheTTSGenerator struct {
	Logger *slog.Logger
}

func (c CacheTTSGenerator) Name() string {
	return "marcus"
}

func (c CacheTTSGenerator) GenerateTTS(input, voice string) ([]byte, error) {
	// Use new cache lookup with fallback to legacy
	provider := "marcus"
	fileName := getFileNameWithFallback(provider, voice, input, c.Logger)

	if !fileIsCached(fileName) {
		return nil, fmt.Errorf("cached file not found for voice '%s'", voice)
	}

	audio, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached TTS file: %v", err)
	}

	c.Logger.Info("found cached TTS file", "file", fileName, "voice", voice)
	return audio, nil
}

func (c CacheTTSGenerator) SupportsVoice(voice string) bool {
	return strings.ToLower(strings.TrimSpace(voice)) == "marcus"
}

func (c CacheTTSGenerator) ListSupportedVoices() ([]string, error) {
	return []string{"marcus"}, nil
}
