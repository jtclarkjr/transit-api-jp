package utils

import (
	kanjikana "github.com/jtclarkjr/kanjikana"
)

func KanjiToRomaji(text string) (string, error) {
	romaji, err := kanjikana.ConvertKanjiToRomaji(text)
	if err != nil {
		return "", err
	}
	return romaji, nil
}
