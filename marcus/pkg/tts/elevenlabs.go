package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
)

type ElevenLabsTTSGenerator struct {
	Logger *slog.Logger
}

func NewElevenLabsTTSGenerator(logger *slog.Logger) (ElevenLabsTTSGenerator, error) {
	gen := ElevenLabsTTSGenerator{Logger: logger}
	_, err := gen.ListSupportedVoices()
	if err != nil {
		return ElevenLabsTTSGenerator{}, err
	}
	return gen, nil
}

func (e ElevenLabsTTSGenerator) Name() string {
	return "elevenlabs"
}

var SupportedElevenLabsVoices sync.Map

func (e ElevenLabsTTSGenerator) GenerateTTS(input, voice string) (string, error) {
	if voice == "" {
		voice = "21m00Tcm4TlvDq8ikWAM"
	}

	e.Logger.Info("requesting ElevenLabs TTS generation", "voice", voice)

	voiceEntryAny, ok := SupportedElevenLabsVoices.Load(voice)
	if !ok {
		return "", fmt.Errorf("unsupported voice: %s", voice)
	}
	voiceEntry := voiceEntryAny.(Voice)

	req, err := e.newElevenLabsRequest("https://api.elevenlabs.io/v1/text-to-dialogue", map[string]interface{}{
		"inputs": []map[string]interface{}{
			{
				"text":     input,
				"voice_id": voiceEntry.VoiceID,
			},
		},
		"model_id": "eleven_v3",
		"settings": map[string]interface{}{
			"stability": 0.5,
		},
	}, http.MethodPost)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.Logger.Error("ElevenLabs TTS request failed", "err", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("elevenlabs api returned status %d: %s", resp.StatusCode, string(body))
		e.Logger.Error("ElevenLabs API non-2xx", "status", resp.StatusCode, "err", err)
		return "", err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.Logger.Error("failed reading ElevenLabs response body", "err", err)
		return "", err
	}

	out := getFileName(input, voice)
	err = os.WriteFile(out, data, 0644)
	if err != nil {
		e.Logger.Error("failed writing ElevenLabs TTS file", "file", out, "err", err)
		return "", err
	}

	e.Logger.Info("downloaded ElevenLabs TTS file", "file", out, "bytes", len(data))
	return out, nil
}

func (e ElevenLabsTTSGenerator) SupportsVoice(voice string) bool {
	_, ok := SupportedElevenLabsVoices.Load(voice)
	return ok
}

func (e ElevenLabsTTSGenerator) ListSupportedVoices() ([]string, error) {
	req, err := e.newElevenLabsRequest("https://api.elevenlabs.io/v2/voices", nil, http.MethodGet)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elevenlabs api returned status %d: %s", resp.StatusCode, string(body))
	}

	list := VoiceListResponse{}
	err = json.Unmarshal(body, &list)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, voice := range list.Voices {
		names = append(names, voice.Name)
		simpleVoice, _, found := strings.Cut(voice.Name, " ")
		if !found {
			continue
		}
		SupportedElevenLabsVoices.Store(strings.ToLower(simpleVoice), voice)
	}

	return names, nil
}

func (e ElevenLabsTTSGenerator) newElevenLabsRequest(url string, body map[string]interface{}, method string) (*http.Request, error) {
	apiKey := os.Getenv("ELEVEN_LABS_API_KEY")
	if apiKey == "" {
		err := fmt.Errorf("ELEVEN_LABS_API_KEY environment variable is not set")
		e.Logger.Error("ElevenLabs API key missing", "err", err)
		return nil, err
	}

	var req *http.Request
	var err error

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			e.Logger.Error("failed to marshal request body", "err", err)
			return nil, err
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
		if err != nil {
			e.Logger.Error("failed to create request", "err", err)
			return nil, err
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			e.Logger.Error("failed to create request", "err", err)
			return nil, err
		}
	}

	req.Header.Set("xi-api-key", apiKey)

	return req, nil
}
