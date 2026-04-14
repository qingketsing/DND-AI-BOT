package search

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	EmbeddingProviderQwen = "qwen"

	defaultEmbeddingBatchSize = 32
	defaultEmbeddingTimeout   = 30 * time.Second
)

var (
	ErrInvalidEmbeddingConfig   = errors.New("invalid embedding config")
	ErrInvalidEmbeddingResponse = errors.New("invalid embedding response")
)

// Embedder 定义统一的文本向量化接口。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingConfig 描述 embedding provider 的运行配置。
type EmbeddingConfig struct {
	Provider  string
	Model     string
	Dim       int
	BatchSize int
	Timeout   time.Duration
	BaseURL   string
	APIKey    string
}

// NormalizeEmbeddingConfig 清理空白并补齐默认值。
func NormalizeEmbeddingConfig(config EmbeddingConfig) EmbeddingConfig {
	config.Provider = strings.TrimSpace(config.Provider)
	config.Model = strings.TrimSpace(config.Model)
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.APIKey = strings.TrimSpace(config.APIKey)
	if config.BatchSize <= 0 {
		config.BatchSize = defaultEmbeddingBatchSize
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultEmbeddingTimeout
	}

	return config
}

// ValidateEmbeddingConfig 校验 embedding provider 的最小配置。
func ValidateEmbeddingConfig(config EmbeddingConfig) error {
	config = NormalizeEmbeddingConfig(config)
	if config.Provider == "" || config.Model == "" || config.BaseURL == "" || config.APIKey == "" {
		return ErrInvalidEmbeddingConfig
	}
	if config.Dim <= 0 || config.BatchSize <= 0 || config.Timeout <= 0 {
		return ErrInvalidEmbeddingConfig
	}

	return nil
}

// EmbedQuery 为单条查询复用批量 embedding 接口。
func EmbedQuery(ctx context.Context, embedder Embedder, text string) ([]float32, error) {
	vectors, err := embedder.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 || len(vectors[0]) == 0 {
		return nil, ErrInvalidEmbeddingResponse
	}

	return append([]float32(nil), vectors[0]...), nil
}
