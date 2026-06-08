package handler

import (
	"crypto/subtle"
	"net/http"
	"os"
)

const appTokenHeader = "X-Transit-App-Token"

func RequireAppToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !hasValidAppToken(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func hasValidAppToken(r *http.Request) bool {
	expected := os.Getenv("OPENAI_PROXY_APP_TOKEN")
	provided := r.Header.Get(appTokenHeader)

	if expected == "" || provided == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
