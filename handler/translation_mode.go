package handler

import (
	"fmt"
	"strings"
)

type translationMode string

const (
	translationModeNone   translationMode = "none"
	translationModeRomaji translationMode = "romaji"
	translationModeOpenAI translationMode = "openai"
)

func parseTranslationMode(lang, aiTranslate string) translationMode {
	if lang != "en" {
		return translationModeNone
	}
	if strings.EqualFold(aiTranslate, "true") {
		return translationModeOpenAI
	}
	return translationModeRomaji
}

func translationCacheVariant(lang string, mode translationMode) string {
	return fmt.Sprintf("%s:%s", lang, mode)
}

func requiresAppTokenForTranslation(mode translationMode) bool {
	return mode == translationModeOpenAI
}
