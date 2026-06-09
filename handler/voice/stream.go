package voice

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
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

	entry, err := speechAudioEntryForSession(session)
	if err != nil {
		log.Printf("Speech cache error: %v", err)
		http.Error(w, "Failed to load speech audio", http.StatusInternalServerError)
		return
	}

	streamSpeechAudioCache(w, r, entry)
}

func speechAudioEntryForSession(session speechSession) (*speechAudioCacheEntry, error) {
	if session.CacheKey != "" {
		if entry, ok := loadSpeechAudioCacheEntry(session.CacheKey); ok {
			ensureSpeechAudioGeneration(entry)
			return entry, nil
		}
	}

	entry, err := getOrCreateSpeechAudioCacheEntry(session.Text, session.Language)
	if err != nil {
		return nil, err
	}
	ensureSpeechAudioGeneration(entry)
	return entry, nil
}

func streamSpeechAudioCache(w http.ResponseWriter, r *http.Request, entry *speechAudioCacheEntry) {
	flusher, _ := w.(http.Flusher)
	wroteHeader := false
	startedAt := time.Now()

	for index := 0; ; index++ {
		chunk, done, err := waitForSpeechAudioChunk(r.Context(), entry, index)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			log.Printf("Speech stream cache error: key=%s chunk=%d err=%v", entry.Key, index+1, err)
			if !wroteHeader {
				http.Error(w, "Failed to generate speech", http.StatusBadGateway)
			}
			return
		}
		if done {
			if !wroteHeader {
				http.Error(w, "Speech audio is empty", http.StatusBadGateway)
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

		if _, err := w.Write(chunk); err != nil {
			log.Printf("Error writing speech stream chunk: key=%s chunk=%d err=%v", entry.Key, index+1, err)
			return
		}
		if flusher != nil {
			flusher.Flush()
		}

		log.Printf("[VOICE STREAM] key=%s chunk=%d bytes=%d total=%s", entry.Key, index+1, len(chunk), time.Since(startedAt))
	}
}

func waitForSpeechAudioChunk(ctx context.Context, entry *speechAudioCacheEntry, index int) ([]byte, bool, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		entry.mu.Lock()
		if index < len(entry.chunks) {
			chunk := append([]byte(nil), entry.chunks[index]...)
			entry.mu.Unlock()
			return chunk, false, nil
		}
		done := entry.done
		err := entry.err
		entry.mu.Unlock()

		if err != nil {
			return nil, true, err
		}
		if done {
			return nil, true, nil
		}

		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-ticker.C:
		}
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
