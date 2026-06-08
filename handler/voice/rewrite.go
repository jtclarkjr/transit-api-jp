package voice

import (
	"context"
	"errors"
	"strings"

	"github.com/openai/openai-go/v3"
)

func rewriteEnglishRouteScript(ctx context.Context, client openai.Client, routeScript string) (string, error) {
	systemPrompt := `Rewrite the route guidance as natural spoken English for text-to-speech.
Convert every Japanese station name, place name, train line name, direction, and landmark into a natural English name or standard romanization.
Keep times, durations, transfer counts, and step order accurate.
Return only the final spoken script. Do not include notes, labels, markdown, or Japanese text unless no English or romanized form is possible.`

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(routeScript),
		},
		Model:               openai.ChatModelGPT4oMini,
		Temperature:         openai.Float(0.2),
		MaxCompletionTokens: openai.Int(900),
	})
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("no route script response")
	}

	text := strings.TrimSpace(completion.Choices[0].Message.Content)
	if text == "" {
		return "", errors.New("empty route script response")
	}

	return text, nil
}
