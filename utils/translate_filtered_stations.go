package utils

import (
	"transit-api/model"
)

// TranslateFilteredStations translates the names of filtered stations to Romaji if the language is English
func TranslateFilteredStations(items []model.FilteredStation) error {
	for i := range items {
		if err := translateString(&items[i].Name); err != nil {
			return err
		}
	}
	return nil
}
