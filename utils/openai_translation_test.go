package utils

import (
	"context"
	"fmt"
	"testing"
)

func TestTranslateStringRefsWithBatchTranslatorDedupesAndCaches(t *testing.T) {
	openAITranslationCache.Clear()

	calls := 0
	var requested []string
	translator := func(_ context.Context, phrases []string) (map[string]string, error) {
		calls++
		requested = append(requested, phrases...)
		translations := make(map[string]string, len(phrases))
		for _, phrase := range phrases {
			translations[phrase] = phrase + " translated"
		}
		return translations, nil
	}

	first := "東京"
	duplicate := "東京"
	empty := " "
	if err := translateStringRefsWithBatchTranslator(context.Background(), []*string{&first, &duplicate, &empty, nil}, translator); err != nil {
		t.Fatalf("translateStringRefsWithBatchTranslator returned error: %v", err)
	}

	if calls != 1 {
		t.Fatalf("translator calls = %d, want 1", calls)
	}
	if len(requested) != 1 || requested[0] != "東京" {
		t.Fatalf("requested = %#v, want only 東京", requested)
	}
	if first != "東京 translated" || duplicate != "東京 translated" {
		t.Fatalf("translations = %q and %q, want cached translated values", first, duplicate)
	}
	if empty != " " {
		t.Fatalf("empty string changed to %q", empty)
	}

	cached := "東京"
	if err := translateStringRefsWithBatchTranslator(context.Background(), []*string{&cached}, func(_ context.Context, _ []string) (map[string]string, error) {
		t.Fatal("translator should not be called for cached source")
		return nil, nil
	}); err != nil {
		t.Fatalf("cached translation returned error: %v", err)
	}
	if cached != "東京 translated" {
		t.Fatalf("cached = %q, want cached translation", cached)
	}
}

func TestTranslateStringRefsWithBatchTranslatorBatchesRequests(t *testing.T) {
	openAITranslationCache.Clear()

	refs := make([]*string, 0, openAITranslationBatchSize+1)
	values := make([]string, 0, openAITranslationBatchSize+1)
	for i := 0; i < openAITranslationBatchSize+1; i++ {
		values = append(values, fmt.Sprintf("駅%d", i))
		refs = append(refs, &values[i])
	}

	var batchSizes []int
	translator := func(_ context.Context, phrases []string) (map[string]string, error) {
		batchSizes = append(batchSizes, len(phrases))
		translations := make(map[string]string, len(phrases))
		for _, phrase := range phrases {
			translations[phrase] = phrase + " translated"
		}
		return translations, nil
	}

	if err := translateStringRefsWithBatchTranslator(context.Background(), refs, translator); err != nil {
		t.Fatalf("translateStringRefsWithBatchTranslator returned error: %v", err)
	}

	if len(batchSizes) != 2 {
		t.Fatalf("batch count = %d, want 2", len(batchSizes))
	}
	if batchSizes[0] != openAITranslationBatchSize || batchSizes[1] != 1 {
		t.Fatalf("batchSizes = %#v, want [%d 1]", batchSizes, openAITranslationBatchSize)
	}
}

func TestTranslateStringRefsWithBatchTranslatorDoesNotCacheIncompleteResults(t *testing.T) {
	openAITranslationCache.Clear()

	source := "新宿"
	err := translateStringRefsWithBatchTranslator(context.Background(), []*string{&source}, func(_ context.Context, _ []string) (map[string]string, error) {
		return map[string]string{}, nil
	})
	if err == nil {
		t.Fatal("expected missing translation error")
	}
	if source != "新宿" {
		t.Fatalf("source changed to %q after failed translation", source)
	}
	if _, ok := openAITranslationCache.Get(openAITranslationCacheKey("新宿")); ok {
		t.Fatal("incomplete translation was cached")
	}
}
