package voice

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

var speechSessions = struct {
	sync.Mutex
	values map[string]speechSession
}{
	values: make(map[string]speechSession),
}

// SpeechSession creates a short-lived speech stream session.
// @Summary Create a speech stream session
// @Description Stores route speech text briefly so iOS can stream playback through AVPlayer.
// @Tags voice
// @Accept json
// @Produce json
// @Param X-Transit-App-Token header string true "Shared app token"
// @Param request body SpeechRequest true "Speech request"
// @Success 200 {object} SpeechSessionResponse "Speech stream session"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized"
// @Router /voice/speech-session [post]
func SpeechSession(w http.ResponseWriter, r *http.Request) {
	text, language, ok := decodeSpeechRequest(w, r)
	if !ok {
		return
	}

	id, entry, err := storeSpeechSession(text, language)
	if err != nil {
		log.Printf("Speech session error: %v", err)
		http.Error(w, "Failed to create speech session", http.StatusInternalServerError)
		return
	}
	ensureSpeechAudioGeneration(entry)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(SpeechSessionResponse{ID: id}); err != nil {
		log.Printf("Error encoding speech session response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func storeSpeechSession(text, language string) (string, *speechAudioCacheEntry, error) {
	now := time.Now()
	entry, err := getOrCreateSpeechAudioCacheEntry(text, language)
	if err != nil {
		return "", nil, err
	}

	speechSessions.Lock()
	defer speechSessions.Unlock()

	cleanupSpeechSessionsLocked(now)
	if len(speechSessions.values) >= maxSpeechSessions {
		return "", nil, errors.New("too many active speech sessions")
	}

	for range 3 {
		id, err := newSpeechSessionID()
		if err != nil {
			return "", nil, err
		}

		if _, exists := speechSessions.values[id]; exists {
			continue
		}

		speechSessions.values[id] = speechSession{
			Text:      text,
			Language:  language,
			CacheKey:  entry.Key,
			ExpiresAt: now.Add(speechSessionTTL),
		}
		return id, entry, nil
	}

	return "", nil, errors.New("failed to allocate unique speech session id")
}

func loadSpeechSession(id string) (speechSession, bool) {
	now := time.Now()

	speechSessions.Lock()
	defer speechSessions.Unlock()

	cleanupSpeechSessionsLocked(now)
	session, ok := speechSessions.values[id]
	if !ok || now.After(session.ExpiresAt) {
		delete(speechSessions.values, id)
		return speechSession{}, false
	}

	return session, true
}

func cleanupSpeechSessionsLocked(now time.Time) {
	for id, session := range speechSessions.values {
		if now.After(session.ExpiresAt) {
			delete(speechSessions.values, id)
		}
	}
}

func newSpeechSessionID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
