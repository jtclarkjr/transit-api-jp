package utils

import (
	"strings"
	"unicode"

	models "transit-api/model"

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

// TranslateTypedTransitResponse translates Japanese text to Romaji in a typed TransitResponse struct
func TranslateTypedTransitResponse(response *models.TransitResponse) error {
	for i := range response.Items {
		item := &response.Items[i]

		// Translate summary names
		if err := translateString(&item.Summary.Start.Name); err != nil {
			return err
		}
		if err := translateString(&item.Summary.Goal.Name); err != nil {
			return err
		}

		// Translate sections
		for j := range item.Sections {
			section := &item.Sections[j]

			// Translate section name
			if err := translateString(&section.Name); err != nil {
				return err
			}

			// Translate line name
			if err := translateString(&section.LineName); err != nil {
				return err
			}

			// Translate transport details if present
			if section.Transport != nil {
				transport := section.Transport

				// Translate transport name
				if err := translateString(&transport.Name); err != nil {
					return err
				}

				// Translate company name
				if err := translateString(&transport.Company.Name); err != nil {
					return err
				}

				// Translate links
				for k := range transport.Links {
					link := &transport.Links[k]
					if err := translateString(&link.Name); err != nil {
						return err
					}
					if err := translateString(&link.Destination.Name); err != nil {
						return err
					}
					if err := translateString(&link.From.Name); err != nil {
						return err
					}
					if err := translateString(&link.To.Name); err != nil {
						return err
					}
				}

				// Translate fare details
				for k := range transport.FareDetail {
					fareDetail := &transport.FareDetail[k]
					if err := translateString(&fareDetail.Start.Name); err != nil {
						return err
					}
					if err := translateString(&fareDetail.Goal.Name); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// translateString is a helper function that translates a single string if it's not empty
func translateString(str *string) error {
	if *str != "" {
		romajiValue, err := KanjiToRomaji(*str)
		if err != nil {
			return err
		}
		*str = CapitalizeFirstLetter(romajiValue)
	}
	return nil
}

// TranslateFilteredStations translates Japanese text to Romaji in filtered station items
func TranslateFilteredStations(items []models.FilteredStation) error {
	for i := range items {
		if err := translateString(&items[i].Name); err != nil {
			return err
		}
	}
	return nil
}
