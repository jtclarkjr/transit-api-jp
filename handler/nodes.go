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
	
	"github.com/jtclarkjr/router-go/middleware"
)

// Permanent cache for station name -> node ID mapping
// Node IDs never change, so no TTL needed
var nodeCache sync.Map

// Used to GET nodeIds for transit request
func fetchNodes(station string, channel chan<- string) {
	// Check cache first
	if cached, ok := nodeCache.Load(station); ok {
		channel <- cached.(string)
		return
	}

	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")

	url := fmt.Sprintf("https://%s/transport_node?word=%s&limit=1", host, station)
	// log.Printf("Fetching node for station: %s, URL: %s", station, url)

	// Rate limit external API call
	middleware.SharedAPIRateLimiter.Wait()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request for station %s: %v", station, err)
		channel <- ""
		return
	}

	request.Header.Add("X-RapidAPI-Key", key)
	request.Header.Add("X-RapidAPI-Host", host)

	response, err := middleware.SharedHTTPClient.Do(request)
	if err != nil {
		log.Printf("Error fetching data for station %s: %v", station, err)
		channel <- ""
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body for station %s: %v", station, err)
			channel <- ""
			return
		}
	}(response.Body)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error reading response for station %s: %v", station, err)
		channel <- ""
		return
	}

	// log.Printf("Response body for station %s: %s", station, string(body))

	var data model.NodeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error unmarshaling response for station %s: %v", station, err)
		channel <- ""
		return
	}

	if len(data.Items) == 0 {
		log.Printf("No items found in response for station %s. Full response: %s", station, string(body))
		channel <- ""
		return
	}

	nodeId := data.Items[0].ID
	if nodeId == "" {
		log.Printf("No node ID found for station %s", station)
		channel <- ""
		return
	}

	// Cache the node ID for future requests
	nodeCache.Store(station, nodeId)

	// log.Printf("Found node ID for station %s: %s", station, nodeId)
	channel <- nodeId
}
