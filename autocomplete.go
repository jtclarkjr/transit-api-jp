package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/gojp/kana"
	"github.com/ikawaha/kagome/tokenizer"
)

type Coord struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Numbering struct {
	Symbol string `json:"symbol"`
	Number string `json:"number"`
}

type Station struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Ruby        string      `json:"ruby"`
	Types       []string    `json:"types"`
	AddressName string      `json:"address_name"`
	AddressCode string      `json:"address_code"`
	Coord       Coord       `json:"coord"`
	Numbering   []Numbering `json:"numbering"`
	Type        string      `json:"type"` // This field will store the first type
}

type FilteredStation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AutocompleteResponse struct {
	Items []Station `json:"items"`
}

type FilteredAutocompleteResponse struct {
	Items []FilteredStation `json:"items"`
}

func kanjiToHiraganaAutocomplete(text string) (string, error) {
	t := tokenizer.New()
	tokens := t.Analyze(text, tokenizer.Normal)
	var hiraganaText string
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		if len(token.Features()) > 7 && token.Features()[7] != "*" {
			hiraganaText += token.Features()[7]
		} else {
			hiraganaText += token.Surface
		}
	}
	return hiraganaText, nil
}

func translateTextToRomajiAutocomplete(text string) (string, error) {
	romajiText := kana.KanaToRomaji(text)
	return romajiText, nil
}

func capitalizeFirstLetter(text string) string {
	// Split by parentheses
	parts := strings.Split(text, "(")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "(")
}

func applyRomajiRules(text string) string {
	replacements := map[string]string{
		"ou": "o",
		"uu": "u",
	}

	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}

	return text
}

func autocomplete(w http.ResponseWriter, r *http.Request) {
	key := os.Getenv("RAPIDAPI_KEY")
	host := os.Getenv("RAPIDAPI_TRANSPORT_HOST")
	lang := r.URL.Query().Get("lang")

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

	// Print the response from RapidAPI
	fmt.Println("Response from RapidAPI:", string(body))

	// Unmarshal the body into the structured response
	var response AutocompleteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		http.Error(w, "Failed to parse JSON response", http.StatusInternalServerError)
		return
	}

	// Filter and transform the response
	var filteredItems []FilteredStation
	for _, item := range response.Items {
		if len(item.Types) > 0 {
			item.Type = item.Types[0]
		}
		filteredItem := FilteredStation{ID: item.ID, Name: item.Name, Type: item.Type}
		filteredItems = append(filteredItems, filteredItem)
	}

	// Translate the name from Japanese to Romaji if lang=en
	if lang == "en" {
		for i, item := range filteredItems {
			hiraganaName, err := kanjiToHiraganaAutocomplete(item.Name)
			if err != nil {
				log.Printf("Error converting Kanji to Hiragana: %v", err)
				http.Error(w, fmt.Sprintf("Failed to convert Kanji to Hiragana: %v", err), http.StatusInternalServerError)
				return
			}

			romajiName, err := translateTextToRomajiAutocomplete(hiraganaName)
			if err != nil {
				log.Printf("Error converting Kana to Romaji: %v", err)
				http.Error(w, fmt.Sprintf("Failed to convert Kana to Romaji: %v", err), http.StatusInternalServerError)
				return
			}

			romajiName = capitalizeFirstLetter(romajiName)
			romajiName = applyRomajiRules(romajiName)

			// Print the original and converted names
			fmt.Printf("Original name: %s, Hiragana name: %s, Romaji name: %s\n", item.Name, hiraganaName, romajiName)
			filteredItems[i].Name = romajiName
		}
	}

	// Marshal the filtered response back to JSON
	filteredResponse := FilteredAutocompleteResponse{Items: filteredItems}
	filteredBody, err := json.Marshal(filteredResponse)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(filteredBody)
}
