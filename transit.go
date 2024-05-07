package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

func fetchNodes(station string, channel chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")

	url := fmt.Sprintf("https://%s/transport_node?word=%s&limit=1", host,
		station,
	)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		channel <- ""
		return
	}

	request.Header.Add("X-RapidAPI-Key", key)
	request.Header.Add("X-RapidAPI-Host", host)

	response, error := http.DefaultClient.Do(request)
	if error != nil {
		channel <- ""
		return
	}
	defer response.Body.Close()

	body, error := io.ReadAll(response.Body)
	if error != nil {
		channel <- ""
		return
	}

	var data map[string]interface{}
	if error := json.Unmarshal(body, &data); error != nil {
		channel <- ""
		return
	}

	items, found := data["items"].([]interface{})
	if !found || len(items) == 0 {
		channel <- ""
		return
	}

	itemMap, ok := items[0].(map[string]interface{})
	if !ok {
		channel <- ""
		return
	}

	nodeId, ok := itemMap["id"].(string)
	if !ok {
		channel <- ""
		return
	}

	channel <- nodeId

	fmt.Println(channel)
}

func transit() http.HandlerFunc {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSIT_HOST")

	return func(w http.ResponseWriter, r *http.Request) {
		startStation := r.URL.Query().Get("start")
		endStation := r.URL.Query().Get("goal")
		startTime := r.URL.Query().Get("start_time")

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
			startTime,
		)

		request, error := http.NewRequest("GET", url, nil)
		if error != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		request.Header.Add("X-RapidAPI-Key", key)
		request.Header.Add("X-RapidAPI-Host", host)

		response, error := http.DefaultClient.Do(request)
		if error != nil {
			http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
			return
		}
		defer response.Body.Close()

		body, error := io.ReadAll(response.Body)
		if error != nil {
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}
