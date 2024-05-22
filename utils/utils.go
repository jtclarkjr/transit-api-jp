package utils

import (
	"strings"
	"unicode"

	kanjikana "github.com/jtclarkjr/kanjikana"
)

func CapitalizeFirstLetter(text string) string {
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

func ApplyRomajiRules(text string) string {
	replacements := map[string]string{
		"ou": "o",
		"uu": "u",
	}

	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}

	return text
}

func KanjiToRomaji(text string) (string, error) {
	romaji, err := kanjikana.ConvertKanjiToRomaji(text)
	if err != nil {
		return "", err
	}
	return romaji, nil
}
