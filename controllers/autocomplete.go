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

	// Filter and transform the response
	var filteredItems []models.FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}
		// Only include items where type is "station"
		if item.Type == "station" {
			filteredItem := models.FilteredStation{ID: item.ID, Name: item.Name, Type: item.Type}
			// Add ruby key for non-en languages
			if lang != "en" {
				filteredItem.Ruby = item.Ruby
			}
			filteredItems = append(filteredItems, filteredItem)
		}
	}

	// Translate the name from Japanese to Romaji if lang=en
	if lang == "en" {
		if err := utils.TranslateFilteredStations(filteredItems); err != nil {
			log.Printf("Error translating station names: %v", err)
			http.Error(w, "Failed to translate station names", http.StatusInternalServerError)
			return
		}
	}

	// Marshal the filtered response back to JSON
	filteredResponse := models.FilteredAutocompleteResponse{Items: filteredItems}
	filteredBody, err := json.Marshal(filteredResponse)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(filteredBody)
}
