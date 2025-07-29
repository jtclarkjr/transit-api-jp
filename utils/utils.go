package utils

import (
	"strings"
	"sync"
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

func KanjiToRomaji(text string) (string, error) {
	romaji, err := kanjikana.ConvertKanjiToRomaji(text)
	if err != nil {
		return "", err
	}
	return romaji, nil
}

func TranslateValueWorker(input <-chan map[string]any, output chan<- error, keysToTranslate []string) {
	for data := range input {
		for _, key := range keysToTranslate {
			keys := strings.Split(key, ".")
			if err := TranslateValueTransit(data, keys); err != nil {
				output <- err
				return
			}
		}
		output <- nil
	}
}

func TranslateJSONValuesTransit(data map[string]any, keysToTranslate []string, numWorkers int) error {
	input := make(chan map[string]any, numWorkers)
	output := make(chan error, numWorkers)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			TranslateValueWorker(input, output, keysToTranslate)
		}()
	}

	input <- data
	close(input)

	wg.Wait()
	close(output)

	for err := range output {
		if err != nil {
			return err
		}
	}

	return nil
}

func TranslateValueTransit(data map[string]any, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	key := keys[0]
	value, found := data[key]

	if !found {
		return nil
	}

	if len(keys) == 1 {
		// We're at the final key in the path
		if strValue, ok := value.(string); ok {
			romajiValue, err := KanjiToRomaji(strValue)
			if err != nil {
				return err
			}
			romajiValue = CapitalizeFirstLetter(romajiValue)
			data[key] = romajiValue
		}
	} else {
		// We're not at the final key, so we need to traverse further
		switch v := value.(type) {
		case map[string]any:
			return TranslateValueTransit(v, keys[1:])
		case []any:
			for _, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					if err := TranslateValueTransit(itemMap, keys[1:]); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
