package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"transit-api/cache"
	"transit-api/model"
	"transit-api/utils"
	
	"github.com/jtclarkjr/router-go/middleware"
)

// Autocomplete cache with 5 minute TTL, max 5000 entries
// Cache key format: "word|lang"
var autocompleteCache = cache.NewLRUCache(5000, 5*time.Minute)

// Single flight to prevent duplicate in-flight requests
var autocompleteSF = middleware.NewSingleFlight()

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

	// Check cache first
	cacheKey := fmt.Sprintf("%s|%s", word, lang)
	if cached, ok := autocompleteCache.Get(cacheKey); ok {
		log.Printf("[CACHE HIT] Autocomplete: key=%s", cacheKey)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_, err := w.Write(cached.([]byte))
		if err != nil {
			log.Printf("Error writing cached response: %v", err)
		}
		return
	}

	// Use single flight to prevent duplicate in-flight requests
	log.Printf("[CACHE MISS] Autocomplete: key=%s, calling API...", cacheKey)
	result, err := autocompleteSF.Do(cacheKey, func() ([]byte, error) {
		return fetchAutocomplete(word, lang, key, host)
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache and return the result
	autocompleteCache.Set(cacheKey, result)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	_, err = w.Write(result)
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// fetchAutocomplete performs the actual API call and processing
func fetchAutocomplete(word, lang, key, host string) ([]byte, error) {
	url := fmt.Sprintf(
		"https://%s/transport_node/autocomplete?word=%s&word_match=prefix",
		host,
		word,
	)

	log.Printf("[API CALL] Autocomplete: word=%s, lang=%s", word, lang)
	
	// Rate limit external API call
	middleware.SharedAPIRateLimiter.Wait()
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-RapidAPI-Key", key)
	req.Header.Add("X-RapidAPI-Host", host)

	res, err := middleware.SharedHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response model.AutocompleteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// If the language is English, translate station names to Romaji
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
					return nil, fmt.Errorf("failed to translate station name: %w", err)
				}
				filteredItemsEn = append(filteredItemsEn, model.FilteredStation{
					ID:   item.ID,
					Name: romajiValue,
					Type: item.Type,
				})
			}
		}
		filteredResponse := model.FilteredAutocompleteResponse{Items: filteredItemsEn}
		return json.Marshal(filteredResponse)
	}

	// Non-English: filter stations only
	var filteredItems []model.FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}
		if item.Type == "station" {
			filteredItems = append(filteredItems, model.FilteredStation{
				ID:   item.ID,
				Name: item.Name,
				Type: item.Type,
			})
		}
	}

	filteredResponse := model.FilteredAutocompleteResponse{Items: filteredItems}
	return json.Marshal(filteredResponse)
}
