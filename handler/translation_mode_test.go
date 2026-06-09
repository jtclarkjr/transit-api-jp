package handler

import (
	"net/http/httptest"
	"testing"
)

func TestParseTranslationMode(t *testing.T) {
	tests := []struct {
		name        string
		lang        string
		aiTranslate string
		want        translationMode
	}{
		{name: "non english ignores ai flag", lang: "", aiTranslate: "true", want: translationModeNone},
		{name: "english defaults to romaji", lang: "en", aiTranslate: "", want: translationModeRomaji},
		{name: "english with ai flag uses openai", lang: "en", aiTranslate: "true", want: translationModeOpenAI},
		{name: "ai flag is case insensitive", lang: "en", aiTranslate: "TRUE", want: translationModeOpenAI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTranslationMode(tt.lang, tt.aiTranslate); got != tt.want {
				t.Fatalf("parseTranslationMode(%q, %q) = %q, want %q", tt.lang, tt.aiTranslate, got, tt.want)
			}
		})
	}
}

func TestTranslationCacheVariantSeparatesModes(t *testing.T) {
	romaji := translationCacheVariant("en", translationModeRomaji)
	openAI := translationCacheVariant("en", translationModeOpenAI)

	if romaji == openAI {
		t.Fatalf("romaji and openai cache variants collided: %q", romaji)
	}
}

func TestAppTokenRequiredOnlyForOpenAITranslation(t *testing.T) {
	if requiresAppTokenForTranslation(translationModeRomaji) {
		t.Fatal("romaji translation should not require app token")
	}
	if !requiresAppTokenForTranslation(translationModeOpenAI) {
		t.Fatal("openai translation should require app token")
	}
}

func TestHasValidAppToken(t *testing.T) {
	t.Setenv("OPENAI_PROXY_APP_TOKEN", "secret")

	req := httptest.NewRequest("GET", "/transit?lang=en&ai_translate=true", nil)
	if hasValidAppToken(req) {
		t.Fatal("request without app token should be invalid")
	}

	req.Header.Set(appTokenHeader, "secret")
	if !hasValidAppToken(req) {
		t.Fatal("request with matching app token should be valid")
	}
}

func TestHasValidAppTokenRequiresTokenForLocalRequests(t *testing.T) {
	t.Setenv("OPENAI_PROXY_APP_TOKEN", "secret")

	req := httptest.NewRequest("GET", "http://localhost:8080/transit?lang=en&ai_translate=true", nil)
	req.RemoteAddr = "127.0.0.1:51234"

	if hasValidAppToken(req) {
		t.Fatal("local request without app token should be invalid")
	}
}
