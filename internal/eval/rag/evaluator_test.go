package rag

import (
	"context"
	"testing"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestEvaluatorRunsLexicalAndHybridSideBySide(t *testing.T) {
	evaluator := NewEvaluator(
		SearcherSet{
			RuleSearcher: &fakeSearcher{results: []retrievalsearch.SearchResult{{ChunkID: "rules-101"}, {ChunkID: "rules-999"}}},
			LoreSearcher: &fakeSearcher{results: []retrievalsearch.SearchResult{{ChunkID: "lore-101"}}},
		},
		SearcherSet{
			RuleSearcher: &fakeSearcher{results: []retrievalsearch.SearchResult{{ChunkID: "rules-999"}, {ChunkID: "rules-101"}}},
			LoreSearcher: &fakeSearcher{results: []retrievalsearch.SearchResult{{ChunkID: "lore-101"}}},
		},
		[]int{1, 3, 5},
		5,
	)

	report, err := evaluator.Evaluate(context.Background(),
		[]Query{{ID: "rules-1", KnowledgeBase: "rules", Query: "潜行规则", QueryType: "semantic"}},
		[]GoldsetEntry{{QueryID: "rules-1", KnowledgeBase: "rules", RelevantChunkIDs: []string{"rules-101"}}},
	)
	if err != nil {
		t.Fatalf("expected evaluation to succeed, got %v", err)
	}
	if report.QueryCount != 1 {
		t.Fatalf("expected query count 1, got %d", report.QueryCount)
	}
	if len(report.Metrics.Records) != 2 {
		t.Fatalf("expected 2 backend records, got %d", len(report.Metrics.Records))
	}
	if got := report.Metrics.Overall["lexical"].RecallAtK[1]; got != 1 {
		t.Fatalf("expected lexical recall@1=1, got %.2f", got)
	}
	if got := report.Metrics.Overall["hybrid"].RecallAtK[1]; got != 0 {
		t.Fatalf("expected hybrid recall@1=0, got %.2f", got)
	}
}

func TestEvaluatorSkipsQueriesWithoutApprovedGoldset(t *testing.T) {
	evaluator := NewEvaluator(
		SearcherSet{
			RuleSearcher: &fakeSearcher{results: []retrievalsearch.SearchResult{{ChunkID: "rules-101"}}},
			LoreSearcher: &fakeSearcher{},
		},
		SearcherSet{
			RuleSearcher: &fakeSearcher{results: []retrievalsearch.SearchResult{{ChunkID: "rules-101"}}},
			LoreSearcher: &fakeSearcher{},
		},
		[]int{1, 3, 5},
		5,
	)

	report, err := evaluator.Evaluate(context.Background(),
		[]Query{
			{ID: "rules-1", KnowledgeBase: "rules", Query: "潜行规则", QueryType: "semantic"},
			{ID: "rules-2", KnowledgeBase: "rules", Query: "法师规则", QueryType: "semantic"},
		},
		[]GoldsetEntry{{QueryID: "rules-1", KnowledgeBase: "rules", RelevantChunkIDs: []string{"rules-101"}}},
	)
	if err != nil {
		t.Fatalf("expected evaluator to skip missing goldset query, got %v", err)
	}
	if report.QueryCount != 1 {
		t.Fatalf("expected query count 1 after skipping missing goldset query, got %d", report.QueryCount)
	}
	if len(report.Metrics.Records) != 2 {
		t.Fatalf("expected only one evaluated query across two backends, got %d records", len(report.Metrics.Records))
	}
}

type fakeSearcher struct {
	results []retrievalsearch.SearchResult
}

func (f *fakeSearcher) Search(ctx context.Context, request retrievalsearch.SearchRequest) ([]retrievalsearch.SearchResult, error) {
	return append([]retrievalsearch.SearchResult(nil), f.results...), nil
}
