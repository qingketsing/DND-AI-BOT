package search

import (
	"context"
	"testing"
	"time"

	postgresstore "DND-AI-BOT/internal/repository/postgres"
)

func TestPostgresIndexMetadataStoreUpsertAndLoad(t *testing.T) {
	t.Parallel()

	db := postgresstore.NewFakeKnowledgePGDB(t, postgresstore.NewFakeKnowledgePGState())
	store := NewPostgresIndexMetadataStore(db)
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	err := store.UpsertIndexMetadata(context.Background(), IndexMetadata{
		KnowledgeBase:     KnowledgeBaseRules,
		EmbeddingProvider: EmbeddingProviderQwen,
		EmbeddingModel:    "Qwen/Qwen3-Embedding-8B",
		EmbeddingDim:      1024,
		BuiltAt:           now,
	})
	if err != nil {
		t.Fatalf("UpsertIndexMetadata() error = %v", err)
	}

	got, err := store.LoadIndexMetadata(context.Background(), KnowledgeBaseRules)
	if err != nil {
		t.Fatalf("LoadIndexMetadata() error = %v", err)
	}
	if got == nil {
		t.Fatal("LoadIndexMetadata() = nil, want metadata")
	}
	if got.KnowledgeBase != KnowledgeBaseRules {
		t.Fatalf("KnowledgeBase = %q, want %q", got.KnowledgeBase, KnowledgeBaseRules)
	}
	if got.EmbeddingProvider != EmbeddingProviderQwen {
		t.Fatalf("EmbeddingProvider = %q, want %q", got.EmbeddingProvider, EmbeddingProviderQwen)
	}
	if got.EmbeddingModel != "Qwen/Qwen3-Embedding-8B" {
		t.Fatalf("EmbeddingModel = %q, want %q", got.EmbeddingModel, "Qwen/Qwen3-Embedding-8B")
	}
	if got.EmbeddingDim != 1024 {
		t.Fatalf("EmbeddingDim = %d, want 1024", got.EmbeddingDim)
	}
	if !got.BuiltAt.Equal(now) {
		t.Fatalf("BuiltAt = %v, want %v", got.BuiltAt, now)
	}
}
