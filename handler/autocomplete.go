package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	models "transit-api/model"
	"transit-api/utils"
)

// KanjiToRomajiFunc is a function type for converting Kanji to Romaji
var KanjiToRomajiFunc = utils.KanjiToRomaji

// AutocompleteHandler allows dependency injection for easier testing
type AutocompleteHandler struct {
	HTTPClient    *http.Client
	KanjiToRomaji func(string) (string, error)
	BaseURL       string // e.g. "https://api.example.com"
}

// NewAutocompleteHandler returns a handler with defaults
func NewAutocompleteHandler() *AutocompleteHandler {
	return &AutocompleteHandler{
		HTTPClient:    http.DefaultClient,
		KanjiToRomaji: utils.KanjiToRomaji,
		BaseURL:       "https://" + os.Getenv("RAPIDAPI_TRANSPORT_HOST"),
	}
}

func (h *AutocompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("RAPIDAPI_KEY")
	lang := r.URL.Query().Get("lang")
	word := r.URL.Query().Get("word")
	url := fmt.Sprintf("%s/transport_node/autocomplete?word=%s&word_match=prefix", h.BaseURL, word)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Add("X-RapidAPI-Key", key)
	// Add host header if using real API
	if h.BaseURL == "https://"+os.Getenv("RAPIDAPI_TRANSPORT_HOST") {
		req.Header.Add("X-RapidAPI-Host", os.Getenv("RAPIDAPI_TRANSPORT_HOST"))
	}

	res, err := h.HTTPClient.Do(req)
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

	var response models.AutocompleteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
		return
	}

	if lang == "en" {
		var filteredItemsEn []models.FilteredStation
		for _, item := range response.Items {
			if len(item.Types) > 0 {
				item.Type = item.Types[0]
			}
			if item.Type == "station" {
				name := item.Ruby
				romajiValue, err := h.KanjiToRomaji(name)
				if err != nil {
					log.Printf("Error translating station name: %v", err)
					http.Error(w, "Failed to translate station names", http.StatusInternalServerError)
					return
				}
				filteredItemsEn = append(filteredItemsEn, models.FilteredStation{
					ID:   item.ID,
					Name: romajiValue,
					Type: item.Type,
				})
			}
		}
		filteredResponse := models.FilteredAutocompleteResponse{Items: filteredItemsEn}
		filteredBody, err := json.Marshal(filteredResponse)
		if err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(filteredBody)
		return
	}

	var filteredItems []models.FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}
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
