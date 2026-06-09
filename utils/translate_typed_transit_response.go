package utils

import (
	"context"
	"transit-api/model"
)

// TranslateTypedTransitResponse translates the names in a TransitResponse to Romaji if the language is English
func TranslateTypedTransitResponse(response *model.TransitResponse) error {
	return translateTypedTransitResponse(response, translateString)
}

// TranslateTypedTransitResponseWithOpenAI translates names in a TransitResponse using OpenAI-backed cached phrase translation.
func TranslateTypedTransitResponseWithOpenAI(ctx context.Context, response *model.TransitResponse) error {
	refs := collectTransitResponseStringRefs(response)
	return TranslateStringsWithOpenAI(ctx, refs...)
}

func translateTypedTransitResponse(response *model.TransitResponse, translate func(*string) error) error {
	for i := range response.Items {
		item := &response.Items[i]

		// Translate summary names
		if err := translate(&item.Summary.Start.Name); err != nil {
			return err
		}
		if err := translate(&item.Summary.Goal.Name); err != nil {
			return err
		}

		// Translate sections
		for j := range item.Sections {
			section := &item.Sections[j]

			// Translate section name
			if err := translate(&section.Name); err != nil {
				return err
			}

			// Translate line name
			if err := translate(&section.LineName); err != nil {
				return err
			}

			// Translate transport details if present
			if section.Transport != nil {
				transport := section.Transport

				// Translate transport name
				if err := translate(&transport.Name); err != nil {
					return err
				}

				// Translate company name
				if err := translate(&transport.Company.Name); err != nil {
					return err
				}

				// Translate links
				for k := range transport.Links {
					link := &transport.Links[k]
					if err := translate(&link.Name); err != nil {
						return err
					}
					if err := translate(&link.Destination.Name); err != nil {
						return err
					}
					if err := translate(&link.From.Name); err != nil {
						return err
					}
					if err := translate(&link.To.Name); err != nil {
						return err
					}
				}

				// Translate fare details
				for k := range transport.FareDetail {
					fareDetail := &transport.FareDetail[k]
					if err := translate(&fareDetail.Start.Name); err != nil {
						return err
					}
					if err := translate(&fareDetail.Goal.Name); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func collectTransitResponseStringRefs(response *model.TransitResponse) []*string {
	refs := make([]*string, 0)

	for i := range response.Items {
		item := &response.Items[i]
		refs = append(refs, &item.Summary.Start.Name, &item.Summary.Goal.Name)

		for j := range item.Sections {
			section := &item.Sections[j]
			refs = append(refs, &section.Name, &section.LineName)

			if section.Transport == nil {
				continue
			}

			transport := section.Transport
			refs = append(refs, &transport.Name, &transport.Company.Name)

			for k := range transport.Links {
				link := &transport.Links[k]
				refs = append(refs, &link.Name, &link.Destination.Name, &link.From.Name, &link.To.Name)
			}

			for k := range transport.FareDetail {
				fareDetail := &transport.FareDetail[k]
				refs = append(refs, &fareDetail.Start.Name, &fareDetail.Goal.Name)
			}
		}
	}

	return refs
}
