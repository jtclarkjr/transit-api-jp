package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"transit-api/models"
	"transit-api/utils"
)

func Autocomplete(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")
	lang := r.URL.Query().Get("lang")

	word := r.URL.Query().Get("word")
	url := fmt.Sprintf(
		"https://%s/transport_node/autocomplete?word=%s&word_match=prefix",
		host,
		word,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Add("X-RapidAPI-Key", key)
	req.Header.Add("X-RapidAPI-Host", host)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to execute request", http.StatusInternalServerError)
		return
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// Print the response from RapidAPI
	// fmt.Println("Response from RapidAPI:", string(body))

	// Unmarshal the body into the structured response
	var response models.AutocompleteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
		return
	}

	// If the language is English, translate station names to Romaji
	// EN value uses ruby key value instead of name since kanji included city names in parenthesis
	// EN still returns the name key with ruby value in Romaji though
	if lang == "en" {
		var filteredItemsEn []map[string]string
		for _, item := range response.Items {
			if len(item.Types) > 0 {
				item.Type = item.Types[0]
			}
			if item.Type == "station" {
				name := item.Ruby
				romajiValue, err := utils.KanjiToRomaji(name)
				if err != nil {
					log.Printf("Error translating station name: %v", err)
					http.Error(w, "Failed to translate station names", http.StatusInternalServerError)
					return
				}
				filteredItemsEn = append(filteredItemsEn, map[string]string{"name": romajiValue, "id": item.ID, "type": item.Type})
			}
		}
		filteredResponse := map[string]interface{}{"items": filteredItemsEn}
		filteredBody, err := json.Marshal(filteredResponse)
		if err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(filteredBody)
		return
	}

	// Non-English: keep previous logic
	var filteredItems []models.FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}

		// Only include items of type "station"
		if item.Type == "station" {
			filteredItem := models.FilteredStation{ID: item.ID, Name: item.Name, Type: item.Type}
			filteredItems = append(filteredItems, filteredItem)
		}
	}

	filteredResponse := models.FilteredAutocompleteResponse{Items: filteredItems}
	filteredBody, err := json.Marshal(filteredResponse)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(filteredBody)
}
