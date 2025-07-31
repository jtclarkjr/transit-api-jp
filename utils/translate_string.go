package utils

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
