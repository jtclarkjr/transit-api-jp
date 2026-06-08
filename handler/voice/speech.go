package voice

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openai/openai-go/v3"
)

// Speech generates speech audio using OpenAI.
// @Summary Generate route speech audio
// @Description Rewrites English route guidance into natural spoken English before generating speech audio.
// @Tags voice
// @Accept json
// @Produce audio/mpeg
// @Param X-Transit-App-Token header string true "Shared app token"
// @Param request body SpeechRequest true "Speech request"
// @Success 200 {file} file "MP3 audio"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized"
// @Failure 502 {string} string "OpenAI upstream error"
// @Router /voice/speech [post]
func Speech(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		http.Error(w, "OPENAI_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	text, language, ok := decodeSpeechRequest(w, r)
	if !ok {
		return
	}

	client := openai.NewClient()
	spokenText := text
	if language == "en" {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		var err error
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

	response, err := newSpeechResponse(ctx, client, spokenText, language)
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

func newSpeechResponse(ctx context.Context, client openai.Client, text, language string) (*http.Response, error) {
	return client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input: text,
		Model: openai.SpeechModelGPT4oMiniTTS,
		Voice: openai.AudioSpeechNewParamsVoiceUnion{
			OfAudioSpeechNewsVoiceString2: openai.String(string(openai.AudioSpeechNewParamsVoiceString2Marin)),
		},
		Instructions:   openai.String(speechInstructions(language)),
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
		StreamFormat:   openai.AudioSpeechNewParamsStreamFormatAudio,
	})
}
