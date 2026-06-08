package voice

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
)

// Transcribe transcribes uploaded audio using OpenAI.
// @Summary Transcribe voice search audio
// @Description Transcribes an uploaded audio file for voice route search.
// @Tags voice
// @Accept multipart/form-data
// @Produce json
// @Param X-Transit-App-Token header string true "Shared app token"
// @Param file formData file true "Audio file"
// @Param language formData string true "Language code: ja or en"
// @Success 200 {object} TranscriptionResponse "Transcribed text"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized"
// @Failure 502 {string} string "OpenAI upstream error"
// @Router /voice/transcribe [post]
func Transcribe(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewEncoder(w).Encode(TranscriptionResponse{Text: text}); err != nil {
		log.Printf("Error encoding transcription response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
