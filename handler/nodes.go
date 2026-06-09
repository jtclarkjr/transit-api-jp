package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"transit-api/model"

	"github.com/jtclarkjr/router-go/middleware"
)

// Permanent cache for station name -> node ID mapping
// Node IDs never change, so no TTL needed
var nodeCache sync.Map

type nodeResult struct {
	nodeID string
	err    error
}

// Used to GET nodeIds for transit request
func fetchNodes(station string, channel chan<- string) {
	nodeID, err := resolveTransitNode(station)
	if err != nil {
		log.Printf("Error resolving node for station %s: %v", station, err)
		channel <- ""
		return
	}
	channel <- nodeID
}

func resolveTransitNode(station string) (string, error) {
	station = strings.TrimSpace(station)
	if station == "" {
		return "", fmt.Errorf("station is required")
	}
	if isNodeID(station) {
		return station, nil
	}

	// Check cache first
	if cached, ok := nodeCache.Load(station); ok {
		return cached.(string), nil
	}
	if looksLikeTranslatedDisplayName(station) {
		return "", fmt.Errorf("station %q appears to be a translated display name; pass a Japanese station name or the id from /autocomplete", station)
	}

	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")

	url := buildTransportNodeURL(host, station)
	// log.Printf("Fetching node for station: %s, URL: %s", station, url)

	// Rate limit external API call
	middleware.SharedAPIRateLimiter.Wait()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Add("X-RapidAPI-Key", key)
	request.Header.Add("X-RapidAPI-Host", host)

	response, err := middleware.SharedHTTPClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("error fetching data: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body for station %s: %v", station, err)
		}
	}(response.Body)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("transport_node upstream returned status %d: %s", response.StatusCode, responsePreview(body))
	}

	// log.Printf("Response body for station %s: %s", station, string(body))

	var data model.NodeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("transport_node returned invalid JSON: %w; body=%s", err, responsePreview(body))
	}

	if len(data.Items) == 0 {
		return "", fmt.Errorf("station %q not found; pass a Japanese station name or the id from /autocomplete", station)
	}

	nodeId := data.Items[0].ID
	if nodeId == "" {
		return "", fmt.Errorf("no node ID found for station %q", station)
	}

	// Cache the node ID for future requests
	nodeCache.Store(station, nodeId)

	// log.Printf("Found node ID for station %s: %s", station, nodeId)
	return nodeId, nil
}

func buildTransportNodeURL(host, station string) string {
	values := url.Values{}
	values.Set("word", station)
	values.Set("limit", "1")

	u := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     "/transport_node",
		RawQuery: values.Encode(),
	}
	return u.String()
}

func isNodeID(value string) bool {
	if len(value) < 5 || len(value) > 16 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func looksLikeTranslatedDisplayName(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || isNodeID(value) {
		return false
	}
	return !containsJapanese(value)
}

func containsJapanese(value string) bool {
	for _, r := range value {
		switch {
		case r >= '\u3040' && r <= '\u309f':
			return true
		case r >= '\u30a0' && r <= '\u30ff':
			return true
		case r >= '\u3400' && r <= '\u9fff':
			return true
		}
	}
	return false
}

func responsePreview(body []byte) string {
	const maxPreviewLength = 300
	preview := strings.TrimSpace(string(body))
	if len(preview) <= maxPreviewLength {
		return preview
	}
	return preview[:maxPreviewLength] + "..."
}
