package rag

import "testing"

func TestRecallAtK(t *testing.T) {
	record := BuildQueryMetrics(
		Query{ID: "rules-1", KnowledgeBase: "rules", QueryType: "semantic"},
		GoldsetEntry{QueryID: "rules-1", KnowledgeBase: "rules", RelevantChunkIDs: []string{"a", "b"}},
		[]string{"x", "a", "b"},
		"lexical",
		[]int{1, 3, 5},
	)

	if got := record.RecallAtK[1]; got != 0 {
		t.Fatalf("expected recall@1 = 0, got %.2f", got)
	}
	if got := record.RecallAtK[3]; got != 1 {
		t.Fatalf("expected recall@3 = 1, got %.2f", got)
	}
}

func TestMRRUsesFirstRelevantRank(t *testing.T) {
	record := BuildQueryMetrics(
		Query{ID: "rules-1", KnowledgeBase: "rules", QueryType: "semantic"},
		GoldsetEntry{QueryID: "rules-1", KnowledgeBase: "rules", RelevantChunkIDs: []string{"b"}},
		[]string{"x", "y", "b"},
		"hybrid",
		[]int{1, 3, 5},
	)

	if got := record.MRR; got != 1.0/3.0 {
		t.Fatalf("expected mrr=1/3, got %.6f", got)
	}
}

func TestAggregateMetricsBuildsGroupedAverages(t *testing.T) {
	records := []QueryEvalRecord{
		{
			QueryID:        "rules-1",
			KnowledgeBase:  "rules",
			QueryType:      "semantic",
			Backend:        "lexical",
			RecallAtK:      map[int]float64{1: 1, 3: 1},
			MRR:            1,
			FirstRelevantRank: 1,
		},
		{
			QueryID:        "lore-1",
			KnowledgeBase:  "lore",
			QueryType:      "exact_name",
			Backend:        "lexical",
			RecallAtK:      map[int]float64{1: 0, 3: 1},
			MRR:            0.5,
			FirstRelevantRank: 2,
		},
	}

	report := BuildMetricsReport([]int{1, 3}, records)

	if got := report.Overall["lexical"].RecallAtK[1]; got != 0.5 {
		t.Fatalf("expected overall recall@1=0.5, got %.2f", got)
	}
	if got := report.ByKnowledgeBase["rules"]["lexical"].MRR; got != 1 {
		t.Fatalf("expected rules lexical mrr=1, got %.2f", got)
	}
	if got := report.ByQueryType["exact_name"]["lexical"].RecallAtK[3]; got != 1 {
		t.Fatalf("expected exact_name lexical recall@3=1, got %.2f", got)
	}
}
