package utils

import (
	"strings"
	"unicode/utf8"
)

var romajiDisplayNameOverrides = map[string]string{
	"スカイツリー":   "Skytree",
	"東京スカイツリー": "Tokyo Skytree",
}

func RomajiDisplayName(source string) (string, error) {
	main, qualifiers := splitDisplayNameQualifiers(source)

	mainDisplay, err := romajiDisplayNameSegment(main, false)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(mainDisplay)

	for _, qualifier := range qualifiers {
		qualifierDisplay, err := romajiDisplayNameSegment(qualifier, true)
		if err != nil {
			return "", err
		}
		if qualifierDisplay == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString("[")
		builder.WriteString(qualifierDisplay)
		builder.WriteString("]")
	}

	return builder.String(), nil
}

func splitDisplayNameQualifiers(source string) (string, []string) {
	var main strings.Builder
	var qualifiers []string

	for i := 0; i < len(source); {
		r, size := utf8.DecodeRuneInString(source[i:])
		closeRune, ok := qualifierCloseRune(r)
		if !ok {
			main.WriteRune(r)
			i += size
			continue
		}

		start := i + size
		endOffset := strings.IndexRune(source[start:], closeRune)
		if endOffset < 0 {
			main.WriteRune(r)
			i += size
			continue
		}

		qualifiers = append(qualifiers, strings.TrimSpace(source[start:start+endOffset]))
		_, closeSize := utf8.DecodeRuneInString(source[start+endOffset:])
		i = start + endOffset + closeSize
	}

	return strings.TrimSpace(main.String()), qualifiers
}

func qualifierCloseRune(open rune) (rune, bool) {
	switch open {
	case '[':
		return ']', true
	case '［':
		return '］', true
	case '(':
		return ')', true
	case '（':
		return '）', true
	default:
		return 0, false
	}
}

func romajiDisplayNameSegment(source string, qualifier bool) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil
	}

	if qualifier {
		if strings.HasSuffix(source, "前") {
			baseDisplay, err := romajiProperName(strings.TrimSuffix(source, "前"))
			if err != nil {
				return "", err
			}
			if baseDisplay == "" {
				return "front", nil
			}
			return baseDisplay + " front", nil
		}
	}

	if strings.HasSuffix(source, "駅") {
		baseDisplay, err := romajiProperName(strings.TrimSuffix(source, "駅"))
		if err != nil {
			return "", err
		}
		if baseDisplay == "" {
			return "Station", nil
		}
		return baseDisplay + " Station", nil
	}

	if strings.HasSuffix(source, "線") {
		baseDisplay, err := romajiProperName(strings.TrimSuffix(source, "線"))
		if err != nil {
			return "", err
		}
		if baseDisplay == "" {
			return "Line", nil
		}
		return baseDisplay + " Line", nil
	}

	return romajiProperName(source)
}

func romajiProperName(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil
	}
	if override, ok := romajiDisplayNameOverrides[source]; ok {
		return override, nil
	}

	romajiValue, err := KanjiToRomaji(source)
	if err != nil {
		return "", err
	}
	return CapitalizeFirstLetter(romajiValue), nil
}
