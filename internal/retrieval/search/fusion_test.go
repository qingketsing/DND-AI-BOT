package search

import "testing"

func TestRRFFusionDeduplicatesOverlapAndCombinesRanks(t *testing.T) {
	t.Parallel()

	fusion := NewRRFFusion(60)
	results := fusion.Fuse(
		[]ScoredCandidate{
			{ChunkID: "chunk-1", Title: "Magic Missile", Content: "spell", KnowledgeBase: KnowledgeBaseRules},
			{ChunkID: "chunk-2", Title: "Wizard", Content: "class", KnowledgeBase: KnowledgeBaseRules},
		},
		[]ScoredCandidate{
			{ChunkID: "chunk-2", Title: "Wizard", Content: "class", KnowledgeBase: KnowledgeBaseRules},
			{ChunkID: "chunk-3", Title: "Spellbook", Content: "feature", KnowledgeBase: KnowledgeBaseRules},
		},
		5,
	)

	if len(results) != 3 {
		t.Fatalf("expected 3 fused results, got %d", len(results))
	}
	if results[0].ChunkID != "chunk-2" {
		t.Fatalf("expected overlapping chunk to rank first, got %s", results[0].ChunkID)
	}
}

func TestRRFFusionHonorsTopK(t *testing.T) {
	t.Parallel()

	fusion := NewRRFFusion(60)
	results := fusion.Fuse(
		[]ScoredCandidate{
			{ChunkID: "chunk-1", Title: "one"},
			{ChunkID: "chunk-2", Title: "two"},
		},
		[]ScoredCandidate{
			{ChunkID: "chunk-3", Title: "three"},
		},
		2,
	)

	if len(results) != 2 {
		t.Fatalf("expected topK truncation to 2, got %d", len(results))
	}
}
