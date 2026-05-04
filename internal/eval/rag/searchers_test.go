package rag

import (
	"testing"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestBuildLexicalSearchers(t *testing.T) {
	searchers, err := BuildLexicalSearchers()
	if err != nil {
		t.Fatalf("expected lexical searchers to build, got %v", err)
	}
	if searchers.RuleSearcher == nil || searchers.LoreSearcher == nil {
		t.Fatalf("expected both lexical searchers, got %+v", searchers)
	}
}

func TestMergeCandidatePoolsDeduplicatesByChunkID(t *testing.T) {
	candidates := MergeCandidatePools(
		[]retrievalsearch.SearchResult{
			{ChunkID: "rules-101", Title: "潜行", Content: "a"},
			{ChunkID: "rules-102", Title: "躲藏", Content: "b"},
		},
		[]retrievalsearch.SearchResult{
			{ChunkID: "rules-102", Title: "躲藏", Content: "b"},
			{ChunkID: "rules-103", Title: "隐匿", Content: "c"},
		},
	)

	if len(candidates) != 3 {
		t.Fatalf("expected 3 merged candidates, got %d", len(candidates))
	}
	if candidates[0].ChunkID != "rules-101" || candidates[2].ChunkID != "rules-103" {
		t.Fatalf("unexpected candidate order: %+v", candidates)
	}
}
