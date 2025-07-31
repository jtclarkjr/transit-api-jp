package utils

import (
	"transit-api/model"
)

// TranslateTypedTransitResponse translates the names in a TransitResponse to Romaji if the language is English
func TranslateTypedTransitResponse(response *model.TransitResponse) error {
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
