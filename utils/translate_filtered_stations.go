package utils

import (
	"transit-api/model"
)

func TranslateFilteredStations(items []model.FilteredStation) error {
	for i := range items {
		if err := translateString(&items[i].Name); err != nil {
			return err
		}
	}
	return nil
}
