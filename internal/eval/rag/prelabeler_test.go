package rag

import "testing"

func TestBuildDraftEntryCopiesCandidateIDs(t *testing.T) {
	entry := buildDraftEntry(
		Query{ID: "rules-1", KnowledgeBase: "rules"},
		[]CandidateChunk{
			{ChunkID: "rules-101"},
			{ChunkID: "rules-102"},
		},
		PrelabelResult{RelevantChunkIDs: []string{"rules-102"}, Reason: "matched"},
	)

	if len(entry.CandidateChunkIDs) != 2 {
		t.Fatalf("expected 2 candidate ids, got %d", len(entry.CandidateChunkIDs))
	}
	if len(entry.PredictedRelevantChunkIDs) != 1 || entry.PredictedRelevantChunkIDs[0] != "rules-102" {
		t.Fatalf("unexpected predicted ids: %+v", entry)
	}
	if entry.ReviewStatus != "draft" {
		t.Fatalf("expected draft review status, got %q", entry.ReviewStatus)
	}
}
