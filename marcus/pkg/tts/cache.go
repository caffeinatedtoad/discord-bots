package tts

import (
	"fmt"
	"log/slog"
	"strings"
)

type CacheTTSGenerator struct {
	Logger *slog.Logger
}

func (c CacheTTSGenerator) Name() string {
	return "marcus"
}

func (c CacheTTSGenerator) GenerateTTS(input, voice string) (string, error) {
	fileName := getFileName(input, voice)
	legacy := legacyFileName(input)

	if !fileIsCached(fileName) && !fileIsCached(legacy) {
		return "", fmt.Errorf("cached file not found: %s %s", fileName, legacy)
	}

	c.Logger.Info("found cached TTS file", "file", fileName, "voice", voice)
	return fileName, nil
}

func (c CacheTTSGenerator) SupportsVoice(voice string) bool {
	return strings.ToLower(strings.TrimSpace(voice)) == "marcus"
}

func (c CacheTTSGenerator) ListSupportedVoices() ([]string, error) {
	return []string{"marcus"}, nil
}
