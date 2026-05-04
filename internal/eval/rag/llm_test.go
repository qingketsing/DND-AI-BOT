package rag

import (
	"context"
	"errors"
	"strings"
	"testing"

	agentclient "DND-AI-BOT/internal/agent/client"
	agentmock "DND-AI-BOT/internal/agent/client/mock"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
)

func TestBuildModelAdapterSupportsMockProvider(t *testing.T) {
	adapter, err := BuildModelAdapter(ModelConfig{Provider: string(agentclient.ProviderMock)})
	if err != nil {
		t.Fatalf("expected mock adapter build to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected adapter")
	}
}

func TestBuildModelAdapterRejectsUnsupportedProvider(t *testing.T) {
	_, err := BuildModelAdapter(ModelConfig{Provider: "unknown"})
	if !errors.Is(err, agentclient.ErrUnsupportedProvider) {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestParsePrelabelResultParsesJSONReply(t *testing.T) {
	result, err := parsePrelabelResult(`{"relevant_chunk_ids":["rules-101","rules-102"],"reason":"ok"}`)
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	if len(result.RelevantChunkIDs) != 2 || result.RelevantChunkIDs[0] != "rules-101" {
		t.Fatalf("unexpected prelabel result: %+v", result)
	}
}

func TestParsePrelabelResultReturnsErrorForInvalidJSON(t *testing.T) {
	_, err := parsePrelabelResult("not json")
	if err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
	if !strings.Contains(err.Error(), "parse prelabel JSON") {
		t.Fatalf("expected parse error context, got %v", err)
	}
}

func TestPrelabelerUsesModelReply(t *testing.T) {
	adapter := agentmock.NewAdapter([]agentruntime.ModelOutput{{Reply: `{"relevant_chunk_ids":["rules-101"],"reason":"title matched"}`}})
	prelabeler := NewPrelabeler(adapter)

	entry, err := prelabeler.Predraft(context.Background(), PrelabelInput{
		Query: Query{
			ID:            "rules-1",
			KnowledgeBase: "rules",
			Query:         "潜行规则",
			QueryType:     "exact_name",
		},
		Candidates: []CandidateChunk{
			{ChunkID: "rules-101", Title: "潜行", Content: "潜行规则正文"},
		},
	})
	if err != nil {
		t.Fatalf("expected prelabel draft to succeed, got %v", err)
	}
	if entry.QueryID != "rules-1" || len(entry.PredictedRelevantChunkIDs) != 1 {
		t.Fatalf("unexpected draft entry: %+v", entry)
	}
}

func TestPrelabelerReturnsDraftEntryOnInvalidModelReply(t *testing.T) {
	adapter := agentmock.NewAdapter([]agentruntime.ModelOutput{{Reply: `not json`}})
	prelabeler := NewPrelabeler(adapter)

	entry, err := prelabeler.Predraft(context.Background(), PrelabelInput{
		Query: Query{
			ID:            "rules-1",
			KnowledgeBase: "rules",
			Query:         "潜行规则",
			QueryType:     "exact_name",
		},
		Candidates: []CandidateChunk{
			{ChunkID: "rules-101", Title: "潜行", Content: "潜行规则正文"},
		},
	})
	if err != nil {
		t.Fatalf("expected invalid model reply to degrade into draft entry, got %v", err)
	}
	if len(entry.PredictedRelevantChunkIDs) != 0 {
		t.Fatalf("expected empty predicted relevant ids, got %+v", entry)
	}
	if !strings.Contains(entry.Notes, "prelabel failed") {
		t.Fatalf("expected failure note, got %+v", entry)
	}
}
