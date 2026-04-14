package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewQwenEmbedderRejectsMissingCredentials(t *testing.T) {
	t.Parallel()

	_, err := NewQwenEmbedder(EmbeddingConfig{
		Provider: "qwen",
		Model:    "qwen-embed-8b",
		Dim:      3,
		Timeout:  time.Second,
	})
	if err != ErrInvalidEmbeddingConfig {
		t.Fatalf("expected ErrInvalidEmbeddingConfig, got %v", err)
	}
}

func TestQwenEmbedderEmbedSuccess(t *testing.T) {
	t.Parallel()

	requests := make([]qwenEmbeddingRequest, 0)
	embedder, err := newQwenEmbedderWithClient(EmbeddingConfig{
		Provider:  "qwen",
		Model:     "qwen-embed-8b",
		Dim:       3,
		BatchSize: 8,
		Timeout:   2 * time.Second,
		BaseURL:   "https://example.com",
		APIKey:    "secret",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("expected /embeddings path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("expected bearer auth, got %q", got)
		}
		defer r.Body.Close()
		var payload qwenEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requests = append(requests, payload)
		if payload.Dimensions != 3 {
			t.Fatalf("expected dimensions=3, got %d", payload.Dimensions)
		}

		switch payload.Input {
		case "one":
			return newHTTPResponse(http.StatusOK, `{"data":[{"embedding":[0.1,0.2,0.3]}]}`), nil
		case "two":
			return newHTTPResponse(http.StatusOK, `{"data":[{"embedding":[0.3,0.2,0.1]}]}`), nil
		default:
			t.Fatalf("unexpected input payload %q", payload.Input)
			return nil, nil
		}
	})})
	if err != nil {
		t.Fatalf("expected embedder init to succeed, got %v", err)
	}

	vectors, err := embedder.Embed(context.Background(), []string{"one", "two"})
	if err != nil {
		t.Fatalf("expected embedding to succeed, got %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != 3 {
		t.Fatalf("expected 3 dimensions, got %d", len(vectors[0]))
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 embedding requests, got %d", len(requests))
	}
}

func TestQwenEmbedderRejectsDimensionMismatch(t *testing.T) {
	t.Parallel()

	embedder, err := newQwenEmbedderWithClient(EmbeddingConfig{
		Provider: "qwen",
		Model:    "qwen-embed-8b",
		Dim:      3,
		Timeout:  2 * time.Second,
		BaseURL:  "https://example.com",
		APIKey:   "secret",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusOK, `{"data":[{"embedding":[0.1,0.2]}]}`), nil
	})})
	if err != nil {
		t.Fatalf("expected embedder init to succeed, got %v", err)
	}

	_, err = embedder.Embed(context.Background(), []string{"one"})
	if err != ErrInvalidEmbeddingResponse {
		t.Fatalf("expected ErrInvalidEmbeddingResponse, got %v", err)
	}
}

func TestQwenEmbedderPropagatesHTTPError(t *testing.T) {
	t.Parallel()

	embedder, err := newQwenEmbedderWithClient(EmbeddingConfig{
		Provider: "qwen",
		Model:    "qwen-embed-8b",
		Dim:      3,
		Timeout:  2 * time.Second,
		BaseURL:  "https://example.com",
		APIKey:   "secret",
	}, &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusBadGateway, "bad gateway"), nil
	})})
	if err != nil {
		t.Fatalf("expected embedder init to succeed, got %v", err)
	}

	_, err = embedder.Embed(context.Background(), []string{"one"})
	if err == nil {
		t.Fatal("expected embedding request to fail on HTTP error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
