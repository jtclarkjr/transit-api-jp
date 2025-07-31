package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"transit-api/model"
	"transit-api/utils"
)

// Transit handles transit route requests
func Transit() http.HandlerFunc {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSIT_HOST")

	return func(w http.ResponseWriter, r *http.Request) {
		// startTime := time.Now()

		startStation := r.URL.Query().Get("start")
		endStation := r.URL.Query().Get("goal")
		startTimeStr := r.URL.Query().Get("start_time")
		lang := r.URL.Query().Get("lang")

		var wg sync.WaitGroup
		startChan := make(chan string, 1)
		endChan := make(chan string, 1)

		wg.Add(2)
		go fetchNodes(startStation, startChan, &wg)
		go fetchNodes(endStation, endChan, &wg)
		wg.Wait()

		startNode := <-startChan
		endNode := <-endChan
		close(startChan)
		close(endChan)

		if startNode == "" || endNode == "" {
			http.Error(w, "Failed to fetch nodes", http.StatusInternalServerError)
			return
		}

		url := fmt.Sprintf(
			"https://%s/route_transit?start=%s&goal=%s&start_time=%s&limit=5",
			host,
			startNode,
			endNode,
			startTimeStr,
		)

		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		request.Header.Add("X-RapidAPI-Key", key)
		request.Header.Add("X-RapidAPI-Host", host)

		response, err := http.DefaultClient.Do(request)
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

		var responseData model.TransitResponse
		if err := json.Unmarshal(body, &responseData); err != nil {
			http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
			return
		}

		// Translate values to romaji if lang=en
		if lang == "en" {
			if err := utils.TranslateTypedTransitResponse(&responseData); err != nil {
				log.Printf("Error translating values: %v", err)
				http.Error(w, "Failed to translate values", http.StatusInternalServerError)
				return
			}
		}

		translatedBody, err := json.Marshal(responseData)
		if err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(translatedBody)
		if err != nil {
			return
		}
	}
}
