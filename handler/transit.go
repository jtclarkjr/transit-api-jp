package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
	"transit-api/cache"
	"transit-api/model"
	"transit-api/utils"

	"github.com/jtclarkjr/router-go/middleware"
)

// Response cache with 5 minute TTL, max 1000 entries
// Cache key format: "start|goal|time_rounded_to_minute|lang:translation_mode"
// Timestamps are rounded to the nearest minute to improve cache hit rate
var responseCache = cache.NewLRUCache(1000, 5*time.Minute)

// Transit handles transit route requests
// @Summary Get transit routes between stations
// @Description Get transit route options between two stations with optional language translation
// @Tags transit
// @Accept json
// @Produce json
// @Param start query string true "Starting station name in Japanese or autocomplete node ID" example("東京駅")
// @Param goal query string true "Destination station name in Japanese or autocomplete node ID" example("新宿駅")
// @Param start_time query string true "Start time in format YYYY-MM-DDTHH:MM:SS" example("2024-01-15T09:00:00")
// @Param lang query string false "Language for response (en for English/Romaji)" example("en")
// @Param ai_translate query bool false "Use OpenAI translation when lang=en; requires X-Transit-App-Token" example(false)
// @Success 200 {object} model.TransitResponse "Successful response with transit routes"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 401 {string} string "Unauthorized - missing or invalid app token for AI translation"
// @Failure 502 {string} string "OpenAI upstream translation error"
// @Failure 500 {string} string "Internal server error"
// @Router /transit [get]
func Transit() http.HandlerFunc {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSIT_HOST")

	return func(w http.ResponseWriter, r *http.Request) {
		// startTime := time.Now()

		startStation := r.URL.Query().Get("start")
		endStation := r.URL.Query().Get("goal")
		startTimeStr := r.URL.Query().Get("start_time")
		lang := r.URL.Query().Get("lang")
		translationMode := parseTranslationMode(lang, r.URL.Query().Get("ai_translate"))

		if startStation == "" {
			http.Error(w, "start is required", http.StatusBadRequest)
			return
		}
		if endStation == "" {
			http.Error(w, "goal is required", http.StatusBadRequest)
			return
		}

		if requiresAppTokenForTranslation(translationMode) && !hasValidAppToken(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Round timestamp to nearest minute for better cache hit rate
		// e.g., 09:05:12 and 09:05:45 both cache as 09:05:00
		parsedTime, err := time.Parse("2006-01-02T15:04:05", startTimeStr)
		if err != nil {
			http.Error(w, "start_time must use format YYYY-MM-DDTHH:MM:SS", http.StatusBadRequest)
			return
		}
		roundedTime := parsedTime.Truncate(time.Minute).Format("2006-01-02T15:04:05")

		// Check response cache first
		cacheKey := fmt.Sprintf("%s|%s|%s|%s", startStation, endStation, roundedTime, translationCacheVariant(lang, translationMode))
		if cached, ok := responseCache.Get(cacheKey); ok {
			log.Printf("[CACHE HIT] Transit: key=%s", cacheKey)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			_, err := w.Write(cached.([]byte))
			if err != nil {
				log.Printf("Error writing cached response: %v", err)
			}
			return
		}

		log.Printf("[CACHE MISS] Transit: key=%s, calling API...", cacheKey)

		var wg sync.WaitGroup
		startChan := make(chan nodeResult, 1)
		endChan := make(chan nodeResult, 1)

		wg.Go(func() {
			nodeID, err := resolveTransitNode(startStation)
			startChan <- nodeResult{nodeID: nodeID, err: err}
		})
		wg.Go(func() {
			nodeID, err := resolveTransitNode(endStation)
			endChan <- nodeResult{nodeID: nodeID, err: err}
		})
		wg.Wait()

		startResult := <-startChan
		endResult := <-endChan
		close(startChan)
		close(endChan)

		if startResult.err != nil {
			log.Printf("Failed to resolve start node: %v", startResult.err)
			http.Error(w, startResult.err.Error(), http.StatusBadRequest)
			return
		}
		if endResult.err != nil {
			log.Printf("Failed to resolve goal node: %v", endResult.err)
			http.Error(w, endResult.err.Error(), http.StatusBadRequest)
			return
		}

		url := buildTransitRouteURL(host, startResult.nodeID, endResult.nodeID, startTimeStr)

		log.Printf("[API CALL] Transit: start=%s, goal=%s", startStation, endStation)

		// Rate limit external API call
		middleware.SharedAPIRateLimiter.Wait()

		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		request.Header.Add("X-RapidAPI-Key", key)
		request.Header.Add("X-RapidAPI-Host", host)

		response, err := middleware.SharedHTTPClient.Do(request)
		if err != nil {
			http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Printf("Error closing response body: %v", err)
				http.Error(w, "Failed to close response body", http.StatusInternalServerError)
				return
			}
		}(response.Body)

		body, err := io.ReadAll(response.Body)
		if err != nil {
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			log.Printf("Transit upstream error: status=%d body=%s", response.StatusCode, string(body))
			http.Error(w, "Transit upstream error", http.StatusBadGateway)
			return
		}

		var upstreamError struct {
			StatusCode int    `json:"status_code"`
			Message    string `json:"message"`
		}
		if err := json.Unmarshal(body, &upstreamError); err == nil && upstreamError.StatusCode != 0 {
			log.Printf("Transit upstream error: status_code=%d message=%s body=%s", upstreamError.StatusCode, upstreamError.Message, string(body))
			http.Error(w, "Transit upstream error", http.StatusBadGateway)
			return
		}

		var responseData model.TransitResponse
		if err := json.Unmarshal(body, &responseData); err != nil {
			http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
			return
		}

		switch translationMode {
		case translationModeRomaji:
			if err := utils.TranslateTypedTransitResponse(&responseData); err != nil {
				log.Printf("Error translating values: %v", err)
				http.Error(w, "Failed to translate values", http.StatusInternalServerError)
				return
			}
		case translationModeOpenAI:
			ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
			err := utils.TranslateTypedTransitResponseWithOpenAI(ctx, &responseData)
			cancel()
			if err != nil {
				log.Printf("OpenAI translation error: %v", err)
				http.Error(w, "Failed to translate values with OpenAI", http.StatusBadGateway)
				return
			}
		}

		translatedBody, err := json.Marshal(responseData)
		if err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}

		// Cache the response
		responseCache.Set(cacheKey, translatedBody)

		// println(string(translatedBody))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "MISS")
		_, err = w.Write(translatedBody)
		if err != nil {
			return
		}
	}
}

func buildTransitRouteURL(host, startNode, endNode, startTime string) string {
	values := url.Values{}
	values.Set("start", startNode)
	values.Set("goal", endNode)
	values.Set("start_time", startTime)
	values.Set("limit", "5")

	u := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     "/route_transit",
		RawQuery: values.Encode(),
	}
	return u.String()
}
