package main

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	kanjikana "github.com/jtclarkjr/kanjikana"
)

func main() {
	loadEnv()
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Get("/transit", transit())
	router.Get("/autocomplete", autocomplete)

	fmt.Println("Starting server on :3000")
	http.ListenAndServe(":3000", router)
}

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("Could not load: %v", err)
	}
}

func capitalizeFirstLetter(text string) string {
	// Split by parentheses
	parts := strings.Split(text, "(")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "(")
}

func applyRomajiRules(text string) string {
	replacements := map[string]string{
		"ou": "o",
		"uu": "u",
	}

	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}

	return text
}

func kanjiToRomaji(text string) (string, error) {
	romaji, err := kanjikana.ConvertKanjiToRomaji(text)
	if err != nil {
		return "", err
	}
	return romaji, nil
}
