package utils

// TranslateFilteredStations translates the names of filtered stations to Romaji if the language is English
func translateString(str *string) error {
	if *str != "" {
		romajiValue, err := RomajiDisplayName(*str)
		if err != nil {
			return err
		}
		*str = romajiValue
	}
	return nil
}
