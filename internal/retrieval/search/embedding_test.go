package search

import (
	"context"
	"errors"
	"testing"
)

func TestEmbedQueryReturnsSingleVector(t *testing.T) {
	t.Parallel()

	embedder := &fakeEmbedder{
		vectors: [][]float32{{0.1, 0.2, 0.3}},
	}

	vector, err := EmbedQuery(context.Background(), embedder, "wizard spellbook")
	if err != nil {
		t.Fatalf("expected query embedding to succeed, got %v", err)
	}
	if len(vector) != 3 {
		t.Fatalf("expected 3-dimensional vector, got %d", len(vector))
	}
	if embedder.lastTexts[0] != "wizard spellbook" {
		t.Fatalf("expected query text to be forwarded, got %q", embedder.lastTexts[0])
	}
}

func TestEmbedQueryRejectsEmptyResults(t *testing.T) {
	t.Parallel()

	_, err := EmbedQuery(context.Background(), &fakeEmbedder{}, "wizard")
	if !errors.Is(err, ErrInvalidEmbeddingResponse) {
		t.Fatalf("expected ErrInvalidEmbeddingResponse, got %v", err)
	}
}

type fakeEmbedder struct {
	vectors   [][]float32
	err       error
	lastTexts []string
}

func (f *fakeEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	f.lastTexts = append([]string(nil), texts...)
	if f.err != nil {
		return nil, f.err
	}
	return append([][]float32(nil), f.vectors...), nil
}
