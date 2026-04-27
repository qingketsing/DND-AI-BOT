package search

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

func TestHybridSearcherRecordsPhaseMetrics(t *testing.T) {
	t.Parallel()

	metrics := observability.NewMetrics(prometheus.NewRegistry())
	searcher := NewHybridSearcher(
		KnowledgeBaseRules,
		&fakeHybridStore{
			ftsResults: []ScoredCandidate{
				{ChunkID: "chunk-1", Title: "Magic Missile", KnowledgeBase: KnowledgeBaseRules},
			},
			vectorResults: []ScoredCandidate{
				{ChunkID: "chunk-1", Title: "Magic Missile", KnowledgeBase: KnowledgeBaseRules},
			},
		},
		&fakeEmbedder{vectors: [][]float32{{0.1, 0.2, 0.3}}},
		NewRRFFusion(60),
		10,
		WithHybridSearchMetrics(metrics),
	)

	_, err := searcher.Search(context.Background(), SearchRequest{Query: "wizard spell", TopK: 5})
	if err != nil {
		t.Fatalf("expected hybrid search to succeed, got %v", err)
	}

	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	body := recorder.Body.String()
	for _, expected := range []string{
		"rag_phase_duration_seconds",
		`knowledge_base="rules"`,
		`phase="fts"`,
		`phase="embedding"`,
		`phase="vector"`,
		`phase="fusion"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in metrics output, got %s", expected, body)
		}
	}
}

func TestHybridSearcherFusesFTSAndVectorResults(t *testing.T) {
	t.Parallel()

	searcher := NewHybridSearcher(
		KnowledgeBaseRules,
		&fakeHybridStore{
			ftsResults: []ScoredCandidate{
				{ChunkID: "chunk-1", Title: "Magic Missile", KnowledgeBase: KnowledgeBaseRules},
			},
			vectorResults: []ScoredCandidate{
				{ChunkID: "chunk-1", Title: "Magic Missile", KnowledgeBase: KnowledgeBaseRules},
				{ChunkID: "chunk-2", Title: "Spellbook", KnowledgeBase: KnowledgeBaseRules},
			},
		},
		&fakeEmbedder{
			vectors: [][]float32{{0.1, 0.2, 0.3}},
		},
		NewRRFFusion(60),
		10,
	)

	results, err := searcher.Search(context.Background(), SearchRequest{Query: "wizard spell", TopK: 5})
	if err != nil {
		t.Fatalf("expected hybrid search to succeed, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 fused results, got %d", len(results))
	}
	if results[0].ChunkID != "chunk-1" {
		t.Fatalf("expected overlapping chunk first, got %s", results[0].ChunkID)
	}
}

func TestHybridSearcherFallsBackToFTSWhenVectorFails(t *testing.T) {
	t.Parallel()

	searcher := NewHybridSearcher(
		KnowledgeBaseRules,
		&fakeHybridStore{
			ftsResults: []ScoredCandidate{
				{ChunkID: "chunk-1", Title: "Magic Missile", KnowledgeBase: KnowledgeBaseRules},
			},
			vectorErr: errors.New("vector down"),
		},
		&fakeEmbedder{vectors: [][]float32{{0.1, 0.2, 0.3}}},
		NewRRFFusion(60),
		10,
	)

	results, err := searcher.Search(context.Background(), SearchRequest{Query: "wizard", TopK: 5})
	if err != nil {
		t.Fatalf("expected fallback to FTS, got %v", err)
	}
	if len(results) != 1 || results[0].ChunkID != "chunk-1" {
		t.Fatalf("expected lexical fallback result, got %+v", results)
	}
}

func TestHybridSearcherFallsBackToVectorWhenFTSFails(t *testing.T) {
	t.Parallel()

	searcher := NewHybridSearcher(
		KnowledgeBaseRules,
		&fakeHybridStore{
			ftsErr: errors.New("fts down"),
			vectorResults: []ScoredCandidate{
				{ChunkID: "chunk-2", Title: "Spellbook", KnowledgeBase: KnowledgeBaseRules},
			},
		},
		&fakeEmbedder{vectors: [][]float32{{0.1, 0.2, 0.3}}},
		NewRRFFusion(60),
		10,
	)

	results, err := searcher.Search(context.Background(), SearchRequest{Query: "spellbook", TopK: 5})
	if err != nil {
		t.Fatalf("expected fallback to vector, got %v", err)
	}
	if len(results) != 1 || results[0].ChunkID != "chunk-2" {
		t.Fatalf("expected vector fallback result, got %+v", results)
	}
}

func TestHybridSearcherReturnsErrorWhenBothBackendsFail(t *testing.T) {
	t.Parallel()

	searcher := NewHybridSearcher(
		KnowledgeBaseRules,
		&fakeHybridStore{
			ftsErr:    errors.New("fts down"),
			vectorErr: errors.New("vector down"),
		},
		&fakeEmbedder{vectors: [][]float32{{0.1, 0.2, 0.3}}},
		NewRRFFusion(60),
		10,
	)

	_, err := searcher.Search(context.Background(), SearchRequest{Query: "spellbook", TopK: 5})
	if err == nil {
		t.Fatal("expected error when both search backends fail")
	}
}

type fakeHybridStore struct {
	ftsResults    []ScoredCandidate
	vectorResults []ScoredCandidate
	ftsErr        error
	vectorErr     error
}

func (f *fakeHybridStore) UpsertChunks(ctx context.Context, chunks []IndexedChunk) error {
	return nil
}

func (f *fakeHybridStore) SearchFTS(ctx context.Context, request HybridSearchRequest) ([]ScoredCandidate, error) {
	if f.ftsErr != nil {
		return nil, f.ftsErr
	}
	return append([]ScoredCandidate(nil), f.ftsResults...), nil
}

func (f *fakeHybridStore) SearchVector(ctx context.Context, request VectorSearchRequest) ([]ScoredCandidate, error) {
	if f.vectorErr != nil {
		return nil, f.vectorErr
	}
	return append([]ScoredCandidate(nil), f.vectorResults...), nil
}
