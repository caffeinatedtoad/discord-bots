package tts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

type TikTokTTSGenerator struct {
	HTTPClient *http.Client
	Logger     *slog.Logger
}

func (t TikTokTTSGenerator) Name() string {
	return "tiktok"
}

func (t TikTokTTSGenerator) GenerateTTS(input, voice string) (string, error) {
	if strings.TrimSpace(voice) == "" {
		voice = DefaultVoice
	}
	t.Logger.Info("requesting TTS generation", "text_speaker", voice)
	req, _ := http.NewRequest(http.MethodPost, "https://api16-normal-useast5.us.tiktokv.com/media/api/text/speech/invoke/", nil)
	req.Header.Set("User-Agent", "com.zhiliaoapp.musically/2022600030 (Linux; U; Android 7.1.2; es_ES; SM-G988N; Build/NRD90M;tt-ok/3.12.13.1)")
	req.Header.Set("Cookie", fmt.Sprintf("sessionid=%s", os.Getenv("magic_key")))

	q := req.URL.Query()
	q.Set("aid", "1233")
	q.Set("speaker_map_type", "0")
	q.Set("text_speaker", voice)
	q.Set("req_text", input)
	req.URL.RawQuery = q.Encode()

	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		t.Logger.Error("TTS request failed", "err", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("tts api returned status %d: %s", resp.StatusCode, string(body))
		t.Logger.Error("TTS API non-2xx", "status", resp.StatusCode, "err", err)
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logger.Error("failed reading TTS response body", "err", err)
		return "", err
	}

	tt := ttsResponse{}
	err = json.Unmarshal(b, &tt)
	if err != nil {
		t.Logger.Error("failed to unmarshal TTS response", "err", err)
		return "", err
	}

	if tt.StatusCode != 0 && strings.ToLower(tt.StatusMsg) != "success" && tt.Data.VStr == "" {
		err := fmt.Errorf("tts api error: code %d message %s", tt.StatusCode, tt.StatusMsg)
		t.Logger.Error("TTS API reported error", "api_message", tt.Message, "err", err)
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(tt.Data.VStr)
	if err != nil {
		t.Logger.Error("failed to decode base64 audio data", "err", err)
		return "", err
	}

	out := getFileName(input, voice)
	err = os.WriteFile(out, data, 0644)
	if err != nil {
		t.Logger.Error("failed writing TTS file", "file", out, "err", err)
		return "", err
	}
	t.Logger.Info("downloaded TTS file", "file", out, "bytes", len(data))
	return out, nil
}

var SupportedTikTokVoices = map[string]string{
	"en_male_narration": "English Male Narration",
}

func (t TikTokTTSGenerator) SupportsVoice(voice string) bool {
	_, ok := SupportedTikTokVoices[voice]
	return ok
}

func (t TikTokTTSGenerator) ListSupportedVoices() ([]string, error) {
	var names []string
	for voice, description := range SupportedTikTokVoices {
		names = append(names, fmt.Sprintf("%s (%s)", voice, description))
	}
	return names, nil
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
