package search

import (
	"context"
	"testing"
	"time"

	postgresstore "DND-AI-BOT/internal/repository/postgres"
)

func TestPostgresHybridSearchStoreUpsertAndSearchFTS(t *testing.T) {
	t.Parallel()

	state := postgresstore.NewFakeKnowledgePGState()
	db := postgresstore.NewFakeKnowledgePGDB(t, state)
	store := NewPostgresHybridSearchStore(db)
	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)

	if err := store.UpsertChunks(context.Background(), []IndexedChunk{
		BuildIndexedChunk(SearchChunk{
			ChunkID:       "rules:1",
			DocumentID:    "doc:1",
			KnowledgeBase: KnowledgeBaseRules,
			SourceType:    "phb",
			DocType:       "spell",
			Title:         "Magic Missile",
			Content:       "Magic Missile always hits.",
			Aliases:       []string{"魔法飞弹"},
			Tags:          []string{"spell"},
			SectionPath:   []string{"Chapter 11"},
			Position:      1,
			ChunkStrategy: "section",
		}, []float32{0.9, 0.1, 0.0}, now),
		BuildIndexedChunk(SearchChunk{
			ChunkID:       "lore:1",
			DocumentID:    "doc:lore",
			KnowledgeBase: KnowledgeBaseLore,
			SourceType:    "setting",
			DocType:       "section",
			Title:         "The City",
			Content:       "The city has slip doors.",
			Aliases:       []string{"城市"},
			Tags:          []string{"lore"},
			SectionPath:   []string{"Setting"},
			Position:      1,
			ChunkStrategy: "paragraph",
		}, []float32{0.1, 0.9, 0.0}, now),
	}); err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	results, err := store.SearchFTS(context.Background(), HybridSearchRequest{
		KnowledgeBase: KnowledgeBaseRules,
		Query:         "magic missile",
		TopK:          5,
	})
	if err != nil {
		t.Fatalf("expected FTS search to succeed, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 FTS result, got %d", len(results))
	}
	if results[0].ChunkID != "rules:1" {
		t.Fatalf("expected rules chunk, got %s", results[0].ChunkID)
	}
}

func TestPostgresHybridSearchStoreSearchVector(t *testing.T) {
	t.Parallel()

	state := postgresstore.NewFakeKnowledgePGState()
	db := postgresstore.NewFakeKnowledgePGDB(t, state)
	store := NewPostgresHybridSearchStore(db)
	now := time.Date(2026, 4, 14, 10, 5, 0, 0, time.UTC)

	if err := store.UpsertChunks(context.Background(), []IndexedChunk{
		{
			ID:            "rules:1",
			KnowledgeBase: KnowledgeBaseRules,
			SourceID:      "doc:1",
			Title:         "Magic Missile",
			Content:       "Magic Missile always hits.",
			Embedding:     []float32{1, 0, 0},
			Metadata:      map[string]any{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "rules:2",
			KnowledgeBase: KnowledgeBaseRules,
			SourceID:      "doc:2",
			Title:         "Spellbook",
			Content:       "Wizards maintain a spellbook.",
			Embedding:     []float32{0, 1, 0},
			Metadata:      map[string]any{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}); err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	results, err := store.SearchVector(context.Background(), VectorSearchRequest{
		KnowledgeBase: KnowledgeBaseRules,
		Query:         "wizard spell",
		QueryVector:   []float32{0.9, 0.1, 0},
		TopK:          2,
	})
	if err != nil {
		t.Fatalf("expected vector search to succeed, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 vector results, got %d", len(results))
	}
	if results[0].ChunkID != "rules:1" {
		t.Fatalf("expected closest chunk first, got %s", results[0].ChunkID)
	}
}
