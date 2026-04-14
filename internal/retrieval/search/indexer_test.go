package search

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBuildEmbeddingTextIncludesTitleAliasesTagsAndContent(t *testing.T) {
	t.Parallel()

	chunk := SearchChunk{
		Title:   "法师准备法术",
		Aliases: []string{"prepared spells", "法术准备"},
		Tags:    []string{"wizard", "spellcasting"},
		Content: "法师在长休后准备法术。",
	}

	got := buildEmbeddingText(chunk)

	for _, want := range []string{
		"title: 法师准备法术",
		"aliases: prepared spells, 法术准备",
		"tags: wizard, spellcasting",
		"content: 法师在长休后准备法术。",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("embedding text missing %q: %s", want, got)
		}
	}
}

func TestBuildIndexedChunksRejectsVectorCountMismatch(t *testing.T) {
	t.Parallel()

	_, err := buildIndexedChunks([]SearchChunk{{ChunkID: "c1"}}, nil, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIndexerBuildIndexEmbedsUpsertsAndWritesMetadata(t *testing.T) {
	t.Parallel()

	store := &fakeHybridSearchStore{}
	metadata := &fakeIndexMetadataStore{}
	embedder := &fakeIndexerEmbedder{vectors: [][]float32{{0.1, 0.2}}}

	indexer := NewIndexer(store, metadata, embedder, EmbeddingConfig{
		Provider: EmbeddingProviderQwen,
		Model:    "Qwen/Qwen3-Embedding-8B",
		Dim:      2,
	})
	indexer.now = func() time.Time { return time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC) }

	err := indexer.BuildIndex(context.Background(), KnowledgeBaseRules, []SearchChunk{{
		ChunkID:       "chunk-1",
		KnowledgeBase: KnowledgeBaseRules,
		DocumentID:    "doc-1",
		Title:         "法师准备法术",
		Content:       "法师在长休后准备法术。",
	}})
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}
	if len(store.upserted) != 1 {
		t.Fatalf("upserted chunks = %d, want 1", len(store.upserted))
	}
	if metadata.last == nil {
		t.Fatal("expected metadata to be written")
	}
	if metadata.last.EmbeddingModel != "Qwen/Qwen3-Embedding-8B" {
		t.Fatalf("metadata model = %q", metadata.last.EmbeddingModel)
	}
}

func TestIndexerBuildIndexWrapsBatchErrorWithChunkRange(t *testing.T) {
	t.Parallel()

	store := &fakeHybridSearchStore{}
	metadata := &fakeIndexMetadataStore{}
	embedder := &fakeIndexerEmbedder{err: errors.New("boom")}

	indexer := NewIndexer(store, metadata, embedder, EmbeddingConfig{
		Provider:  EmbeddingProviderQwen,
		Model:     "Qwen/Qwen3-Embedding-8B",
		Dim:       2,
		BatchSize: 2,
	})

	err := indexer.BuildIndex(context.Background(), KnowledgeBaseRules, []SearchChunk{
		{ChunkID: "chunk-1", KnowledgeBase: KnowledgeBaseRules, Content: "one"},
		{ChunkID: "chunk-2", KnowledgeBase: KnowledgeBaseRules, Content: "two"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "chunk_range=0:2") {
		t.Fatalf("error = %q, want chunk range context", err)
	}
}

func TestIndexerBuildIndexUpsertsPerBatch(t *testing.T) {
	t.Parallel()

	store := &fakeHybridSearchStore{}
	metadata := &fakeIndexMetadataStore{}
	embedder := &fakeIndexerEmbedder{
		vectors: [][]float32{
			{0.1, 0.2},
			{0.2, 0.3},
			{0.3, 0.4},
		},
	}

	indexer := NewIndexer(store, metadata, embedder, EmbeddingConfig{
		Provider:  EmbeddingProviderQwen,
		Model:     "Qwen/Qwen3-Embedding-8B",
		Dim:       2,
		BatchSize: 2,
	})

	err := indexer.BuildIndex(context.Background(), KnowledgeBaseRules, []SearchChunk{
		{ChunkID: "chunk-1", KnowledgeBase: KnowledgeBaseRules, DocumentID: "doc-1", Content: "one"},
		{ChunkID: "chunk-2", KnowledgeBase: KnowledgeBaseRules, DocumentID: "doc-2", Content: "two"},
		{ChunkID: "chunk-3", KnowledgeBase: KnowledgeBaseRules, DocumentID: "doc-3", Content: "three"},
	})
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}
	if len(store.upsertCalls) != 2 {
		t.Fatalf("upsert calls = %d, want 2", len(store.upsertCalls))
	}
	if len(store.upsertCalls[0]) != 2 {
		t.Fatalf("first upsert size = %d, want 2", len(store.upsertCalls[0]))
	}
	if len(store.upsertCalls[1]) != 1 {
		t.Fatalf("second upsert size = %d, want 1", len(store.upsertCalls[1]))
	}
}

type fakeIndexMetadataStore struct {
	last *IndexMetadata
}

func (s *fakeIndexMetadataStore) UpsertIndexMetadata(_ context.Context, metadata IndexMetadata) error {
	copyValue := metadata
	s.last = &copyValue
	return nil
}

func (s *fakeIndexMetadataStore) LoadIndexMetadata(_ context.Context, _ string) (*IndexMetadata, error) {
	return s.last, nil
}

type fakeHybridSearchStore struct {
	upserted    []IndexedChunk
	upsertCalls [][]IndexedChunk
}

func (s *fakeHybridSearchStore) UpsertChunks(_ context.Context, chunks []IndexedChunk) error {
	copyChunk := append([]IndexedChunk(nil), chunks...)
	s.upserted = append(s.upserted, copyChunk...)
	s.upsertCalls = append(s.upsertCalls, copyChunk)
	return nil
}

func (s *fakeHybridSearchStore) SearchFTS(_ context.Context, _ HybridSearchRequest) ([]ScoredCandidate, error) {
	return nil, nil
}

func (s *fakeHybridSearchStore) SearchVector(_ context.Context, _ VectorSearchRequest) ([]ScoredCandidate, error) {
	return nil, nil
}

type fakeIndexerEmbedder struct {
	vectors [][]float32
	err     error
}

func (e *fakeIndexerEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.vectors[:len(texts)], nil
}
