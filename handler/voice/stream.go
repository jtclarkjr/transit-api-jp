package voice

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
)

// SpeechStream streams generated speech audio for an existing session.
// @Summary Stream speech audio
// @Description Streams route speech as smaller MP3 chunks for faster playback startup.
// @Tags voice
// @Produce audio/mpeg
// @Param X-Transit-App-Token header string true "Shared app token"
// @Param id query string true "Speech session id"
// @Success 200 {file} file "MP3 audio stream"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Speech session not found"
// @Failure 502 {string} string "OpenAI upstream error"
// @Router /voice/speech-stream [get]
func SpeechStream(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		http.Error(w, "OPENAI_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "Speech session id is required", http.StatusBadRequest)
		return
	}

	session, ok := loadSpeechSession(id)
	if !ok {
		http.Error(w, "Speech session not found", http.StatusNotFound)
		return
	}

	streamSpeechChunks(w, r, session)
}

func streamSpeechChunks(w http.ResponseWriter, r *http.Request, session speechSession) {
	chunks := speechChunks(session.Text)
	if len(chunks) == 0 {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	client := openai.NewClient()
	flusher, _ := w.(http.Flusher)
	wroteHeader := false
	startedAt := time.Now()

	for index, chunk := range chunks {
		chunkStartedAt := time.Now()
		spokenChunk := chunk
		if session.Language == "en" {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			var err error
			spokenChunk, err = rewriteEnglishRouteScript(ctx, client, chunk)
			cancel()
			if err != nil {
				log.Printf("OpenAI route script chunk rewrite error: chunk=%d/%d err=%v", index+1, len(chunks), err)
				if !wroteHeader {
					http.Error(w, "Failed to rewrite speech text", http.StatusBadGateway)
				}
				return
			}
		}

		if len([]rune(spokenChunk)) > maxSpeechInputCharacters {
			log.Printf("Speech chunk too long after rewrite: chunk=%d/%d chars=%d", index+1, len(chunks), len([]rune(spokenChunk)))
			if !wroteHeader {
				http.Error(w, "Speech text is too long", http.StatusBadRequest)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		response, err := newSpeechResponse(ctx, client, spokenChunk, session.Language)
		if err != nil {
			cancel()
			log.Printf("OpenAI speech stream error: chunk=%d/%d err=%v", index+1, len(chunks), err)
			if !wroteHeader {
				http.Error(w, "Failed to generate speech", http.StatusBadGateway)
			}
			return
		}

		if !wroteHeader {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusOK)
			wroteHeader = true
			if flusher != nil {
				flusher.Flush()
			}
		}

		_, copyErr := io.Copy(w, response.Body)
		closeErr := response.Body.Close()
		cancel()
		if copyErr != nil {
			log.Printf("Error writing speech stream chunk: chunk=%d/%d err=%v", index+1, len(chunks), copyErr)
			return
		}
		if closeErr != nil {
			log.Printf("Error closing speech stream chunk: chunk=%d/%d err=%v", index+1, len(chunks), closeErr)
		}
		if flusher != nil {
			flusher.Flush()
		}

		log.Printf("[VOICE STREAM] chunk=%d/%d source_chars=%d elapsed=%s total=%s", index+1, len(chunks), len([]rune(chunk)), time.Since(chunkStartedAt), time.Since(startedAt))
	}
}

func speechChunks(text string) []string {
	sentences := splitSpeechSentences(text)
	chunks := make([]string, 0, len(sentences))
	var current strings.Builder
	currentLength := 0

	flush := func() {
		chunk := strings.TrimSpace(current.String())
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		current.Reset()
		currentLength = 0
	}

	for _, sentence := range sentences {
		sentenceLength := len([]rune(sentence))
		if sentenceLength > maxSpeechStreamChunkCharacters {
			flush()
			chunks = append(chunks, hardSplitSpeechText(sentence)...)
			continue
		}

		separatorLength := 0
		if currentLength > 0 {
			separatorLength = 1
		}

		if currentLength > 0 && currentLength+separatorLength+sentenceLength > maxSpeechStreamChunkCharacters {
			flush()
		}

		if currentLength > 0 {
			current.WriteString(" ")
			currentLength++
		}
		current.WriteString(sentence)
		currentLength += sentenceLength
	}

	flush()
	return chunks
}

func splitSpeechSentences(text string) []string {
	sentences := []string{}
	var current strings.Builder

	for _, char := range text {
		current.WriteRune(char)
		if isSpeechSentenceBoundary(char) {
			sentence := strings.TrimSpace(current.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			current.Reset()
		}
	}

	sentence := strings.TrimSpace(current.String())
	if sentence != "" {
		sentences = append(sentences, sentence)
	}

	return sentences
}

func hardSplitSpeechText(text string) []string {
	runes := []rune(text)
	chunks := []string{}
	for start := 0; start < len(runes); start += maxSpeechStreamChunkCharacters {
		end := start + maxSpeechStreamChunkCharacters
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}
	return chunks
}

func isSpeechSentenceBoundary(char rune) bool {
	switch char {
	case '.', '?', '!', '。', '？', '！':
		return true
	default:
		return false
	}
}
