package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"transit-api/model"
)

// Used to GET nodeIds for transit request
func fetchNodes(station string, channel chan<- string) {

	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")

	url := fmt.Sprintf("https://%s/transport_node?word=%s&limit=1", host, station)
	// log.Printf("Fetching node for station: %s, URL: %s", station, url)

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

	// log.Printf("Found node ID for station %s: %s", station, nodeId)
	channel <- nodeId
}
