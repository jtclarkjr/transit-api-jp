package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
)

const (
	maxSpeechInputCharacters = 4096
	maxTranscriptionBytes    = 20 * 1024 * 1024
)

type VoiceSpeechRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

type VoiceTranscriptionResponse struct {
	Text string `json:"text"`
}

// VoiceSpeech generates speech audio using OpenAI.
// @Summary Generate route speech audio
// @Description Rewrites English route guidance into natural spoken English before generating speech audio.
// @Tags voice
// @Accept json
// @Produce audio/mpeg
// @Param X-Transit-App-Token header string true "Shared app token"
// @Param request body VoiceSpeechRequest true "Speech request"
// @Success 200 {file} file "MP3 audio"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized"
// @Failure 502 {string} string "OpenAI upstream error"
// @Router /voice/speech [post]
func VoiceSpeech(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		http.Error(w, "OPENAI_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	var req VoiceSpeechRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	language, err := normalizeVoiceLanguage(req.Language)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len([]rune(text)) > maxSpeechInputCharacters {
		http.Error(w, "Text is too long", http.StatusBadRequest)
		return
	}

	client := openai.NewClient()
	spokenText := text
	if language == "en" {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		spokenText, err = rewriteEnglishRouteScript(ctx, client, text)
		cancel()
		if err != nil {
			log.Printf("OpenAI route script rewrite error: %v", err)
			http.Error(w, "Failed to rewrite speech text", http.StatusBadGateway)
			return
		}
	}

	if len([]rune(spokenText)) > maxSpeechInputCharacters {
		http.Error(w, "Speech text is too long", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input: spokenText,
		Model: openai.SpeechModelGPT4oMiniTTS,
		Voice: openai.AudioSpeechNewParamsVoiceUnion{
			OfAudioSpeechNewsVoiceString2: openai.String(string(openai.AudioSpeechNewParamsVoiceString2Marin)),
		},
		Instructions:   openai.String(speechInstructions(language)),
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	})
	if err != nil {
		log.Printf("OpenAI speech error: %v", err)
		http.Error(w, "Failed to generate speech", http.StatusBadGateway)
		return
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			log.Printf("Error closing speech response body: %v", closeErr)
		}
	}()

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := io.Copy(w, response.Body); err != nil {
		log.Printf("Error writing speech response: %v", err)
	}
}

// VoiceTranscribe transcribes uploaded audio using OpenAI.
// @Summary Transcribe voice search audio
// @Description Transcribes an uploaded audio file for voice route search.
// @Tags voice
// @Accept multipart/form-data
// @Produce json
// @Param X-Transit-App-Token header string true "Shared app token"
// @Param file formData file true "Audio file"
// @Param language formData string true "Language code: ja or en"
// @Success 200 {object} VoiceTranscriptionResponse "Transcribed text"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized"
// @Failure 502 {string} string "OpenAI upstream error"
// @Router /voice/transcribe [post]
func VoiceTranscribe(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		http.Error(w, "OPENAI_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxTranscriptionBytes)
	if err := r.ParseMultipartForm(maxTranscriptionBytes); err != nil {
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	language, err := normalizeTranscriptionLanguage(r.FormValue("language"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing uploaded file: %v", closeErr)
		}
	}()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/m4a"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	client := openai.NewClient()
	transcription, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:     openai.File(file, header.Filename, contentType),
		Model:    openai.AudioModelWhisper1,
		Language: openai.String(language),
	})
	if err != nil {
		log.Printf("OpenAI transcription error: %v", err)
		http.Error(w, "Failed to transcribe audio", http.StatusBadGateway)
		return
	}

	text := strings.TrimSpace(transcription.Text)
	if text == "" {
		http.Error(w, "Empty transcription", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(VoiceTranscriptionResponse{Text: text}); err != nil {
		log.Printf("Error encoding transcription response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func rewriteEnglishRouteScript(ctx context.Context, client openai.Client, routeScript string) (string, error) {
	systemPrompt := `Rewrite the route guidance as natural spoken English for text-to-speech.
Convert every Japanese station name, place name, train line name, direction, and landmark into a natural English name or standard romanization.
Keep times, durations, transfer counts, and step order accurate.
Return only the final spoken script. Do not include notes, labels, markdown, or Japanese text unless no English or romanized form is possible.`

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(routeScript),
		},
		Model:               openai.ChatModelGPT5Mini,
		Temperature:         openai.Float(0.2),
		MaxCompletionTokens: openai.Int(900),
	})
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("no route script response")
	}

	text := strings.TrimSpace(completion.Choices[0].Message.Content)
	if text == "" {
		return "", errors.New("empty route script response")
	}

	return text, nil
}

func normalizeVoiceLanguage(language string) (string, error) {
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
