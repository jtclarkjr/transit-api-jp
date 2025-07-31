package utils

// TranslateFilteredStations translates the names of filtered stations to Romaji if the language is English
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
