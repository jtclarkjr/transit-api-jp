package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type TransitAgentRequest struct {
	Prompt string `json:"prompt"`
}

type TransitAgentResponse struct {
	StartStation string `json:"start_station"`
	EndStation   string `json:"end_station"`
}

// TransitAgent handles transit agent requests using OpenAI
// @Summary Find nearest stations using AI
// @Description Uses OpenAI to determine the nearest start and end stations based on a location prompt
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
	// Set a 10 second timeout for the API call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shorter, more direct prompt for faster processing
	systemPrompt := `Find nearest train station to the location and suggest destination. Return JSON:
{"start_station":"駅名","end_station":"駅名"}
Both in Japanese with 駅 suffix.`

	// Optimize for speed: low temperature, limited tokens, structured output
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(req.Prompt),
		},
		Model:               openai.ChatModelGPT4oMini,
		Temperature:         openai.Float(0.0), // Deterministic for faster responses
		MaxCompletionTokens: openai.Int(100),   // Limit output tokens (uses max_completion_tokens)
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &[]shared.ResponseFormatJSONObjectParam{shared.NewResponseFormatJSONObjectParam()}[0],
		},
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
