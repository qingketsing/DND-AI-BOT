package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QwenEmbedder 通过 Qwen embedding API 生成向量。
type QwenEmbedder struct {
	config EmbeddingConfig
	client *http.Client
}

func NewQwenEmbedder(config EmbeddingConfig) (*QwenEmbedder, error) {
	return newQwenEmbedderWithClient(config, nil)
}

func newQwenEmbedderWithClient(config EmbeddingConfig, client *http.Client) (*QwenEmbedder, error) {
	config = NormalizeEmbeddingConfig(config)
	if err := ValidateEmbeddingConfig(config); err != nil {
		return nil, err
	}
	if config.Provider != EmbeddingProviderQwen {
		return nil, ErrInvalidEmbeddingConfig
	}

	return &QwenEmbedder{
		config: config,
		client: resolveEmbeddingHTTPClient(client, config.Timeout),
	}, nil
}

func (e *QwenEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	requestBody, err := json.Marshal(qwenEmbeddingRequest{
		Model: e.config.Model,
		Input: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal qwen embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.config.BaseURL+"/embeddings", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create qwen embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qwen embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("qwen embedding request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload qwenEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode qwen embedding response: %w", err)
	}
	if len(payload.Data) != len(texts) {
		return nil, ErrInvalidEmbeddingResponse
	}

	vectors := make([][]float32, 0, len(payload.Data))
	for _, item := range payload.Data {
		if len(item.Embedding) != e.config.Dim {
			return nil, ErrInvalidEmbeddingResponse
		}
		vectors = append(vectors, append([]float32(nil), item.Embedding...))
	}

	return vectors, nil
}

type qwenEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type qwenEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func resolveEmbeddingHTTPClient(client *http.Client, timeout time.Duration) *http.Client {
	if client == nil {
		return &http.Client{Timeout: timeout}
	}
	if client.Timeout <= 0 {
		client.Timeout = timeout
	}
	return client
}
