package utils

import (
	"strings"
	"unicode"
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
