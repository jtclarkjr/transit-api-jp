package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/openai/openai-go/v3"
)

type TransitAgentRequest struct {
	Prompt string `json:"prompt"`
}

type TransitAgentResponse struct {
	StartStation string `json:"start_station"`
	EndStation   string `json:"end_station"`
}

// TransitAgent handles transit agent requests using OpenAI GPT-5
// @Summary Find nearest stations using AI
// @Description Uses OpenAI GPT-5 to determine the nearest start and end stations based on a location prompt
// @Tags transit-agent
// @Accept json
// @Produce json
// @Param request body TransitAgentRequest true "Transit agent request with location prompt"
// @Success 200 {object} TransitAgentResponse "Successful response with start and end stations in Japanese"
// @Failure 400 {string} string "Bad request - missing or invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /transit-agent [post]
func TransitAgent(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		http.Error(w, "OPENAI_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	var req TransitAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	client := openai.NewClient()
	ctx := context.Background()

	systemPrompt := `You are a Japanese transit assistant. Given a location or place mentioned by the user, find the nearest train/subway station to that place as the start station, and suggest a logical destination station based on common transit patterns in Japan.

Your response must be ONLY a valid JSON object with no additional text, in this exact format:
{
  "start_station": "駅名（日本語）",
  "end_station": "駅名（日本語）"
}

Rules:
- Both station names must be in Japanese (kanji/hiragana)
- Include 駅 suffix for station names
- Choose realistic, commonly used stations in Japan
- For the start station, find the nearest station to the mentioned location
- For the end station, suggest a logical destination (major hub, tourist spot, or business district)
- Return ONLY the JSON object, no explanations or additional text`

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(req.Prompt),
		},
		Model: openai.ChatModelGPT5Nano,
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		http.Error(w, "Failed to get response from OpenAI", http.StatusInternalServerError)
		return
	}

	if len(completion.Choices) == 0 {
		http.Error(w, "No response from OpenAI", http.StatusInternalServerError)
		return
	}

	content := completion.Choices[0].Message.Content
	log.Printf("OpenAI response: %s", content)

	var response TransitAgentResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		log.Printf("Failed to parse OpenAI response: %v", err)
		http.Error(w, "Failed to parse AI response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
