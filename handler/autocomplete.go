package handler

import (
	"context"
	"encoding/json"
	"errors"
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

// Autocomplete cache with 30 day TTL, max 5000 entries
// Cache key format: "word|lang:translation_mode"
// Station names don't change, so long TTL is appropriate
var autocompleteCache = cache.NewLRUCache(5000, 30*24*time.Hour)

// Single flight to prevent duplicate in-flight requests
var autocompleteSF = middleware.NewSingleFlight()

var errOpenAITranslation = errors.New("openai translation failed")

// Autocomplete handles autocomplete requests for station names
// @Summary Get station name suggestions
// @Description Get autocomplete suggestions for station names with optional language translation
// @Tags autocomplete
// @Accept json
// @Produce json
// @Param word query string true "Search word for station names" example("東京")
// @Param lang query string false "Language for response (en for English/Romaji)" example("en")
// @Param ai_translate query bool false "Use OpenAI translation when lang=en; requires X-Transit-App-Token" example(false)
// @Success 200 {object} model.FilteredAutocompleteResponse "Successful response with station suggestions"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized - missing or invalid app token for AI translation"
// @Failure 502 {string} string "OpenAI upstream translation error"
// @Failure 500 {string} string "Internal server error"
// @Router /autocomplete [get]
func Autocomplete(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")
	lang := r.URL.Query().Get("lang")
	translationMode := parseTranslationMode(lang, r.URL.Query().Get("ai_translate"))

	word := r.URL.Query().Get("word")

	if requiresAppTokenForTranslation(translationMode) && !hasValidAppToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s|%s", word, translationCacheVariant(lang, translationMode))
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
		return fetchAutocomplete(r.Context(), word, translationMode, key, host)
	})

	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errOpenAITranslation) {
			status = http.StatusBadGateway
		}
		http.Error(w, err.Error(), status)
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
func fetchAutocomplete(ctx context.Context, word string, translationMode translationMode, key, host string) ([]byte, error) {
	url := fmt.Sprintf(
		"https://%s/transport_node/autocomplete?word=%s&word_match=prefix",
		host,
		word,
	)

	log.Printf("[API CALL] Autocomplete: word=%s, translation_mode=%s", word, translationMode)

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

	var filteredItems []model.FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}
		if item.Type == "station" {
			name := item.Name
			if translationMode == translationModeRomaji {
				var err error
				name, err = utils.RomajiDisplayName(item.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to translate station name: %w", err)
				}
			}
			filteredItems = append(filteredItems, model.FilteredStation{
				ID:   item.ID,
				Name: name,
				Type: item.Type,
			})
		}
	}

	if translationMode == translationModeOpenAI {
		refs := make([]*string, 0, len(filteredItems))
		for i := range filteredItems {
			refs = append(refs, &filteredItems[i].Name)
		}

		translationCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := utils.TranslateStringsWithOpenAI(translationCtx, refs...)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errOpenAITranslation, err)
		}
	}

	filteredResponse := model.FilteredAutocompleteResponse{Items: filteredItems}
	return json.Marshal(filteredResponse)
}
