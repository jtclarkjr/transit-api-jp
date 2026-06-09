package voice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openai/openai-go/v3"
)

type speechAudioCacheEntry struct {
	Key       string
	Text      string
	Language  string
	ExpiresAt time.Time

	mu      sync.Mutex
	chunks  [][]byte
	started bool
	done    bool
	err     error

	sizeBytes int64
}

var speechAudioCache = struct {
	sync.Mutex
	values     map[string]*speechAudioCacheEntry
	totalBytes int64
}{
	values: make(map[string]*speechAudioCacheEntry),
}

func getOrCreateSpeechAudioCacheEntry(text, language string) (*speechAudioCacheEntry, error) {
	now := time.Now()
	key := speechAudioCacheKey(text, language)

	speechAudioCache.Lock()
	defer speechAudioCache.Unlock()

	cleanupSpeechAudioCacheLocked(now)

	if entry, ok := speechAudioCache.values[key]; ok {
		if entry.hasFailed() {
			removeSpeechAudioCacheEntryLocked(key, entry)
		} else {
			entry.ExpiresAt = now.Add(speechAudioCacheTTL)
			return entry, nil
		}
	}

	if len(speechAudioCache.values) >= maxSpeechCacheEntries {
		trimSpeechAudioCacheLocked(now, key)
	}
	if len(speechAudioCache.values) >= maxSpeechCacheEntries {
		return nil, errors.New("too many speech cache entries")
	}

	entry := &speechAudioCacheEntry{
		Key:       key,
		Text:      text,
		Language:  language,
		ExpiresAt: now.Add(speechAudioCacheTTL),
	}
	speechAudioCache.values[key] = entry
	return entry, nil
}

func loadSpeechAudioCacheEntry(key string) (*speechAudioCacheEntry, bool) {
	now := time.Now()

	speechAudioCache.Lock()
	defer speechAudioCache.Unlock()

	cleanupSpeechAudioCacheLocked(now)
	entry, ok := speechAudioCache.values[key]
	if !ok || now.After(entry.ExpiresAt) {
		if ok {
			removeSpeechAudioCacheEntryLocked(key, entry)
		}
		return nil, false
	}

	entry.ExpiresAt = now.Add(speechAudioCacheTTL)
	return entry, true
}

func ensureSpeechAudioGeneration(entry *speechAudioCacheEntry) {
	entry.mu.Lock()
	if entry.started {
		entry.mu.Unlock()
		return
	}
	entry.started = true
	entry.mu.Unlock()

	go generateSpeechAudio(entry)
}

func generateSpeechAudio(entry *speechAudioCacheEntry) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		completeSpeechAudioGeneration(entry, errors.New("OPENAI_API_KEY not configured"))
		return
	}

	chunks := speechChunks(entry.Text)
	if len(chunks) == 0 {
		completeSpeechAudioGeneration(entry, errors.New("text is required"))
		return
	}

	client := openai.NewClient()
	startedAt := time.Now()

	for index, chunk := range chunks {
		chunkStartedAt := time.Now()
		audio, err := generateSpeechAudioChunk(client, chunk, entry.Language)
		if err != nil {
			log.Printf("OpenAI speech prefetch error: key=%s chunk=%d/%d err=%v", entry.Key, index+1, len(chunks), err)
			completeSpeechAudioGeneration(entry, err)
			return
		}

		appendSpeechAudioChunk(entry, audio)
		log.Printf("[VOICE PREFETCH] key=%s chunk=%d/%d source_chars=%d bytes=%d elapsed=%s total=%s", entry.Key, index+1, len(chunks), len([]rune(chunk)), len(audio), time.Since(chunkStartedAt), time.Since(startedAt))
	}

	completeSpeechAudioGeneration(entry, nil)
}

func generateSpeechAudioChunk(client openai.Client, chunk, language string) ([]byte, error) {
	spokenChunk := chunk
	if language == "en" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var err error
		spokenChunk, err = rewriteEnglishRouteScript(ctx, client, chunk)
		cancel()
		if err != nil {
			return nil, err
		}
	}

	if len([]rune(spokenChunk)) > maxSpeechInputCharacters {
		return nil, errors.New("speech text is too long")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	response, err := newSpeechResponse(ctx, client, spokenChunk, language)
	if err != nil {
		cancel()
		return nil, err
	}

	audio, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	cancel()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(audio) == 0 {
		return nil, errors.New("empty speech audio response")
	}

	return audio, nil
}

func appendSpeechAudioChunk(entry *speechAudioCacheEntry, audio []byte) {
	entry.mu.Lock()
	entry.chunks = append(entry.chunks, audio)
	entry.mu.Unlock()

	sizeDelta := int64(len(audio))
	atomic.AddInt64(&entry.sizeBytes, sizeDelta)

	speechAudioCache.Lock()
	speechAudioCache.totalBytes += sizeDelta
	trimSpeechAudioCacheLocked(time.Now(), entry.Key)
	speechAudioCache.Unlock()
}

func completeSpeechAudioGeneration(entry *speechAudioCacheEntry, err error) {
	entry.mu.Lock()
	entry.err = err
	entry.done = true
	entry.mu.Unlock()
}

func speechAudioCacheKey(text, language string) string {
	normalizedText := strings.Join(strings.Fields(text), " ")
	sum := sha256.Sum256([]byte(language + "\n" + normalizedText))
	return hex.EncodeToString(sum[:])
}

func cleanupSpeechAudioCacheLocked(now time.Time) {
	for key, entry := range speechAudioCache.values {
		if now.After(entry.ExpiresAt) {
			removeSpeechAudioCacheEntryLocked(key, entry)
		}
	}
	if speechAudioCache.totalBytes > maxSpeechCacheBytes {
		trimSpeechAudioCacheLocked(now, "")
	}
}

func trimSpeechAudioCacheLocked(now time.Time, protectedKey string) {
	for (len(speechAudioCache.values) > maxSpeechCacheEntries || speechAudioCache.totalBytes > maxSpeechCacheBytes) && len(speechAudioCache.values) > 0 {
		var oldestKey string
		var oldestEntry *speechAudioCacheEntry
		for key, entry := range speechAudioCache.values {
			if key == protectedKey {
				continue
			}
			if !entry.isRemovable() {
				continue
			}
			if oldestEntry == nil || entry.ExpiresAt.Before(oldestEntry.ExpiresAt) {
				oldestKey = key
				oldestEntry = entry
			}
		}
		if oldestEntry == nil {
			return
		}
		removeSpeechAudioCacheEntryLocked(oldestKey, oldestEntry)
	}

	for key, entry := range speechAudioCache.values {
		if key != protectedKey && now.After(entry.ExpiresAt) {
			removeSpeechAudioCacheEntryLocked(key, entry)
		}
	}
}

func removeSpeechAudioCacheEntryLocked(key string, entry *speechAudioCacheEntry) {
	delete(speechAudioCache.values, key)
	speechAudioCache.totalBytes -= atomic.LoadInt64(&entry.sizeBytes)
	if speechAudioCache.totalBytes < 0 {
		speechAudioCache.totalBytes = 0
	}
}

func (entry *speechAudioCacheEntry) isRemovable() bool {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.done || entry.err != nil
}

func (entry *speechAudioCacheEntry) hasFailed() bool {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.err != nil
}
