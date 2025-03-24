package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"transit-api/utils"
)

// Used to GET nodeIds for transit request
func fetchNodes(station string, channel chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")

	url := fmt.Sprintf("https://%s/transport_node?word=%s&limit=1", host, station)
	log.Printf("Fetching node for station: %s, URL: %s", station, url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request for station %s: %v", station, err)
		channel <- ""
		return
	}

	request.Header.Add("X-RapidAPI-Key", key)
	request.Header.Add("X-RapidAPI-Host", host)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("Error fetching data for station %s: %v", station, err)
		channel <- ""
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error reading response for station %s: %v", station, err)
		channel <- ""
		return
	}

	log.Printf("Response body for station %s: %s", station, string(body))

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error unmarshaling response for station %s: %v", station, err)
		channel <- ""
		return
	}

	items, found := data["items"].([]interface{})
	if !found || len(items) == 0 {
		log.Printf("No items found in response for station %s. Full response: %s", station, string(body))
		channel <- ""
		return
	}

	itemMap, ok := items[0].(map[string]interface{})
	if !ok {
		log.Printf("Error processing item map for station %s", station)
		channel <- ""
		return
	}

	nodeId, ok := itemMap["id"].(string)
	if !ok {
		log.Printf("No node ID found for station %s", station)
		channel <- ""
		return
	}

	log.Printf("Found node ID for station %s: %s", station, nodeId)
	channel <- nodeId
}

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
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		var responseData map[string]interface{}
		if err := json.Unmarshal(body, &responseData); err != nil {
			http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
			return
		}

		// Translate values to romaji if lang=en
		if lang == "en" {
			keysToTranslate := []string{
				"items.sections.coord.name",
				"items.sections.transport.company.name",
				"items.sections.transport.fare_detail.goal.name",
				"items.sections.transport.fare_detail.start.name",
				"items.sections.transport.links.destination.name",
				"items.sections.transport.links.from.name",
				"items.sections.transport.links.to.name",
				"items.sections.transport.name",
				"items.sections.line_name",
				"items.sections.name",
				"items.summary.goal.name",
				"items.summary.start.name",
			}

			if err := utils.TranslateJSONValuesTransit(responseData, keysToTranslate, 10); err != nil {
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
		w.Write(translatedBody)

		// elapsedTime := time.Since(startTime)
		// log.Printf("Request processed in %s", elapsedTime)
	}
}
