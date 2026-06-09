package voice

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	maxSpeechInputCharacters       = 4096
	maxSpeechStreamChunkCharacters = 650
	maxSpeechSessions              = 200
	maxSpeechCacheEntries          = 100
	maxSpeechCacheBytes            = 50 * 1024 * 1024
	maxTranscriptionBytes          = 20 * 1024 * 1024
	speechSessionTTL               = 5 * time.Minute
	speechAudioCacheTTL            = 30 * time.Minute
)

type SpeechRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

type SpeechSessionResponse struct {
	ID string `json:"id"`
}

type TranscriptionResponse struct {
	Text string `json:"text"`
}

type speechSession struct {
	Text      string
	Language  string
	CacheKey  string
	ExpiresAt time.Time
}

func decodeSpeechRequest(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	var req SpeechRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return "", "", false
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return "", "", false
	}

	language, err := normalizeSpeechLanguage(req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", "", false
	}

	if len([]rune(text)) > maxSpeechInputCharacters {
		http.Error(w, "Text is too long", http.StatusBadRequest)
		return "", "", false
	}

	return text, language, true
}

func normalizeSpeechLanguage(language string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "ja", "ja-jp", "japanese":
		return "ja", nil
	case "en", "en-us", "english":
		return "en", nil
	default:
		return "", errors.New("Language must be ja or en")
	}
}

func normalizeTranscriptionLanguage(language string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "ja", "ja-jp", "japanese":
		return "ja", nil
	case "en", "en-us", "english":
		return "en", nil
	default:
		return "", errors.New("Language must be ja or en")
	}
}

func speechInstructions(language string) string {
	if language == "en" {
		return "Speak naturally in English with a warm, clear transit guide style. Sound like you are explaining directions to a traveler, not reading a list. Use relaxed pacing and clear station names."
	}

	return "Speak naturally in Japanese with a warm, clear transit guide style. Sound like you are explaining directions to a traveler, not reading a list. Use relaxed pacing and clear station names."
}
