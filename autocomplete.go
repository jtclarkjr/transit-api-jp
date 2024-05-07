package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Station struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Type  string   `json:"type"`            // This field will store the first type
	Types []string `json:"types,omitempty"` // Temporary field to capture the types array
}

type AutocompleteResponse struct {
	Items []Station `json:"items"`
}

func autocomplete(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")

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

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the body into the structured response
	var response AutocompleteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
		return
	}

	// Process each item to store only the first type
	for i, item := range response.Items {
		if len(item.Types) > 0 {
			response.Items[i].Type = item.Types[0]
		}
		// Clear the Types slice to ensure it doesn't get included in the final JSON output
		response.Items[i].Types = nil
	}

	// Marshal the filtered response back to JSON
	filteredBody, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(filteredBody)
}
