package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"transit-api/cache"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

const (
	openAITranslationCacheVersion = "ja-en:v2:"
	openAITranslationBatchSize    = 50
)

var openAITranslationCache = cache.NewLRUCache(10000, 30*24*time.Hour)

type phraseBatchTranslator func(context.Context, []string) (map[string]string, error)

// TranslateStringsWithOpenAI translates Japanese transit names with a cached OpenAI-backed phrase translator.
func TranslateStringsWithOpenAI(ctx context.Context, refs ...*string) error {
	return translateStringRefsWithBatchTranslator(ctx, refs, translatePhrasesWithOpenAI)
}

func translateStringRefsWithBatchTranslator(ctx context.Context, refs []*string, translator phraseBatchTranslator) error {
	translations := make(map[string]string)
	uncached := make([]string, 0)
	seen := make(map[string]struct{})

	for _, ref := range refs {
		if ref == nil || strings.TrimSpace(*ref) == "" {
			continue
		}

		source := *ref
		if cached, ok := openAITranslationCache.Get(openAITranslationCacheKey(source)); ok {
			translated, ok := cached.(string)
			if ok {
				translations[source] = translated
				continue
			}
		}

		if _, ok := seen[source]; ok {
			continue
		}
		seen[source] = struct{}{}
		uncached = append(uncached, source)
	}

	for start := 0; start < len(uncached); start += openAITranslationBatchSize {
		end := start + openAITranslationBatchSize
		if end > len(uncached) {
			end = len(uncached)
		}

		batch := uncached[start:end]
		batchTranslations, err := translator(ctx, batch)
		if err != nil {
			return err
		}

		for _, source := range batch {
			translated, ok := batchTranslations[source]
			if !ok || strings.TrimSpace(translated) == "" {
				return fmt.Errorf("missing OpenAI translation for %q", source)
			}
			translations[source] = translated
		}

		for _, source := range batch {
			openAITranslationCache.Set(openAITranslationCacheKey(source), translations[source])
		}
	}

	for _, ref := range refs {
		if ref == nil || strings.TrimSpace(*ref) == "" {
			continue
		}
		if translated, ok := translations[*ref]; ok {
			*ref = translated
		}
	}

	return nil
}

func translatePhrasesWithOpenAI(ctx context.Context, phrases []string) (map[string]string, error) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return nil, errors.New("OPENAI_API_KEY not configured")
	}
	if len(phrases) == 0 {
		return map[string]string{}, nil
	}

	payload, err := json.Marshal(phrases)
	if err != nil {
		return nil, err
	}

	systemPrompt := `Convert Japanese transit station, line, company, destination, and place names into polished English-facing romanized display names.
Do not translate the meaning of proper names. Romanize the proper-name part and only convert generic transit suffixes or locators when natural: 駅 -> Station, 線 -> Line, 方面 -> bound for/toward, 前 -> front.
Keep bracketed or parenthetical qualifiers separate with a space before the bracket. Do not concatenate bracketed text into the main name.
For example, 日の出駅 must become Hinode Station, not Sunrise Station or Hinodeeki.
For example, 押上[スカイツリー前] must become Oshiage [Skytree front], not Oshiagesukaitsuri-mae.
Return only a JSON object where each key is one exact input string and each value is its romanized display name.
Convert every input string. Preserve route numbers, platform labels, train service names, and direction names accurately.`

	client := openai.NewClient()
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(string(payload)),
		},
		Model:               openai.ChatModelGPT4oMini,
		Temperature:         openai.Float(0),
		MaxCompletionTokens: openai.Int(2000),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &[]shared.ResponseFormatJSONObjectParam{shared.NewResponseFormatJSONObjectParam()}[0],
		},
	})
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) == 0 {
		return nil, errors.New("no OpenAI translation response")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return nil, errors.New("empty OpenAI translation response")
	}

	var translations map[string]string
	if err := json.Unmarshal([]byte(content), &translations); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI translation response: %w", err)
	}

	return translations, nil
}

func openAITranslationCacheKey(source string) string {
	return openAITranslationCacheVersion + source
}
