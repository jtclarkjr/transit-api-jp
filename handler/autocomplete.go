package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"transit-api/model"
	"transit-api/utils"
)

// Autocomplete handles autocomplete requests for station names
// @Summary Get station name suggestions
// @Description Get autocomplete suggestions for station names with optional language translation
// @Tags autocomplete
// @Accept json
// @Produce json
// @Param word query string true "Search word for station names" example("東京")
// @Param lang query string false "Language for response (en for English/Romaji)" example("en")
// @Success 200 {object} model.FilteredAutocompleteResponse "Successful response with station suggestions"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /autocomplete [get]
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
			http.Error(w, "Failed to close response body", http.StatusInternalServerError)
			return
		}
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// Print the response from RapidAPI
	// fmt.Println("Response from RapidAPI:", string(body))

	// Unmarshal the body into the structured response
	var response model.AutocompleteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
		return
	}

	// If the language is English, translate station names to Romaji
	// EN value uses ruby key value instead of name since kanji included city names in parentheses
	// EN still returns the name key with ruby value in Romaji though
	if lang == "en" {
		var filteredItemsEn []model.FilteredStation
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
				filteredItemsEn = append(filteredItemsEn, model.FilteredStation{
					ID:   item.ID,
					Name: romajiValue,
					Type: item.Type,
				})
			}
		}
		filteredResponse := model.FilteredAutocompleteResponse{Items: filteredItemsEn}
		filteredBody, err := json.Marshal(filteredResponse)
		if err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(filteredBody)
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
		return
	}

	// Non-English: keep previous logic
	var filteredItems []model.FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}

		// Only include items of type "station"
		if item.Type == "station" {
			filteredItem := model.FilteredStation{ID: item.ID, Name: item.Name, Type: item.Type}
			filteredItems = append(filteredItems, filteredItem)
		}
	}

	filteredResponse := model.FilteredAutocompleteResponse{Items: filteredItems}
	filteredBody, err := json.Marshal(filteredResponse)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(filteredBody)
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
