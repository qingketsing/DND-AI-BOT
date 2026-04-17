package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestSearchRulesToolCallReturnsSearchResults(t *testing.T) {
	searcher := &fakeKnowledgeSearcher{
		results: []retrievalsearch.SearchResult{
			{
				ChunkID:       "rules:phb:class:wizard:0001",
				DocumentID:    "rules:phb:class:wizard",
				KnowledgeBase: retrievalsearch.KnowledgeBaseRules,
				Title:         "法师",
				Content:       "法师使用奥术施法。",
				Score:         26,
			},
		},
	}
	tool := NewSearchRulesTool(searcher)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"query":"法师","top_k":3}`),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}

	if searcher.lastRequest.Query != "法师" {
		t.Fatalf("expected query to be passed through, got %q", searcher.lastRequest.Query)
	}
	if searcher.lastRequest.TopK != 3 {
		t.Fatalf("expected top_k to be passed through, got %d", searcher.lastRequest.TopK)
	}

	result, ok := output.Content.(searchKnowledgeResult)
	if !ok {
		t.Fatalf("expected searchKnowledgeResult output, got %T", output.Content)
	}
	if result.KnowledgeBase != retrievalsearch.KnowledgeBaseRules {
		t.Fatalf("expected rules knowledge base, got %q", result.KnowledgeBase)
	}
	if len(result.Results) != 1 || result.Results[0].ChunkID != "rules:phb:class:wizard:0001" {
		t.Fatalf("expected one rules result, got %+v", result.Results)
	}
}

func TestSearchRulesToolCallRejectsInvalidInput(t *testing.T) {
	tool := NewSearchRulesTool(&fakeKnowledgeSearcher{})

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"query":1}`),
	})
	if !errors.Is(err, ErrInvalidToolInput) {
		t.Fatalf("expected ErrInvalidToolInput, got %v", err)
	}
}

func TestSearchLoreToolCallReturnsSearchResults(t *testing.T) {
	searcher := &fakeKnowledgeSearcher{
		results: []retrievalsearch.SearchResult{
			{
				ChunkID:       "lore:default-setting:dead-world:0001",
				DocumentID:    "lore:default-setting:dead-world",
				KnowledgeBase: retrievalsearch.KnowledgeBaseLore,
				Title:         "死寂的世界",
				Content:       "这里只有城市。",
				Score:         18,
			},
		},
	}
	tool := NewSearchLoreTool(searcher)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"query":"城市"}`),
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}

	if searcher.lastRequest.TopK != 5 {
		t.Fatalf("expected default top_k to be 5, got %d", searcher.lastRequest.TopK)
	}

	result, ok := output.Content.(searchKnowledgeResult)
	if !ok {
		t.Fatalf("expected searchKnowledgeResult output, got %T", output.Content)
	}
	if result.KnowledgeBase != retrievalsearch.KnowledgeBaseLore {
		t.Fatalf("expected lore knowledge base, got %q", result.KnowledgeBase)
	}
	if len(result.Results) != 1 || result.Results[0].ChunkID != "lore:default-setting:dead-world:0001" {
		t.Fatalf("expected one lore result, got %+v", result.Results)
	}
}

func TestSearchLoreToolCallReturnsDegradedResultWhenSearcherFails(t *testing.T) {
	searcher := &fakeKnowledgeSearcher{err: errors.New("search failed")}
	tool := NewSearchLoreTool(searcher)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"query":"城市"}`),
	})
	if err != nil {
		t.Fatalf("expected degraded result instead of error, got %v", err)
	}

	result, ok := output.Content.(searchKnowledgeResult)
	if !ok {
		t.Fatalf("expected searchKnowledgeResult output, got %T", output.Content)
	}
	if !result.Degraded {
		t.Fatalf("expected degraded search result, got %+v", result)
	}
	if !strings.Contains(result.Message, "知识库检索暂时不可用") {
		t.Fatalf("expected degraded message, got %q", result.Message)
	}
}

type fakeKnowledgeSearcher struct {
	lastRequest retrievalsearch.SearchRequest
	results     []retrievalsearch.SearchResult
	err         error
}

func (f *fakeKnowledgeSearcher) Search(ctx context.Context, request retrievalsearch.SearchRequest) ([]retrievalsearch.SearchResult, error) {
	f.lastRequest = request
	if f.err != nil {
		return nil, f.err
	}
	return append([]retrievalsearch.SearchResult(nil), f.results...), nil
}
