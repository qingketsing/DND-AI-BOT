package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidIndexBuildInput = errors.New("invalid index build input")

type ChunkSource interface {
	Load(ctx context.Context) ([]SearchChunk, error)
}

type Indexer struct {
	store         HybridSearchStore
	metadataStore IndexMetadataStore
	embedder      Embedder
	config        EmbeddingConfig
	now           func() time.Time
}

func NewIndexer(
	store HybridSearchStore,
	metadataStore IndexMetadataStore,
	embedder Embedder,
	config EmbeddingConfig,
) *Indexer {
	return &Indexer{
		store:         store,
		metadataStore: metadataStore,
		embedder:      embedder,
		config:        NormalizeEmbeddingConfig(config),
		now:           time.Now,
	}
}

func (i *Indexer) BuildIndex(ctx context.Context, knowledgeBase string, chunks []SearchChunk) error {
	if knowledgeBase != KnowledgeBaseRules && knowledgeBase != KnowledgeBaseLore {
		return ErrInvalidIndexBuildInput
	}
	if len(chunks) == 0 {
		return i.metadataStore.UpsertIndexMetadata(ctx, IndexMetadata{
			KnowledgeBase:     knowledgeBase,
			EmbeddingProvider: i.config.Provider,
			EmbeddingModel:    i.config.Model,
			EmbeddingDim:      i.config.Dim,
			BuiltAt:           i.now(),
		})
	}

	texts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		texts = append(texts, buildEmbeddingText(chunk))
	}

	batchSize := i.config.BatchSize
	if batchSize <= 0 {
		batchSize = defaultEmbeddingBatchSize
	}

	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batchVectors, err := i.embedder.Embed(ctx, texts[start:end])
		if err != nil {
			return fmt.Errorf("embed chunk_range=%d:%d: %w", start, end, err)
		}
		indexed, err := buildIndexedChunks(chunks[start:end], batchVectors, i.now())
		if err != nil {
			return err
		}
		if err := i.store.UpsertChunks(ctx, indexed); err != nil {
			return err
		}

		// Progress Bar
		percent := float64(end) / float64(len(texts)) * 100
		barLength := 40
		completedLength := int(float64(barLength) * float64(end) / float64(len(texts)))
		bar := strings.Repeat("=", completedLength) + strings.Repeat("-", barLength-completedLength)
		fmt.Printf("\rEmbedding %s: [%s] %d/%d (%.1f%%)", knowledgeBase, bar, end, len(texts), percent)
	}
	fmt.Println() // Print a newline when done

	return i.metadataStore.UpsertIndexMetadata(ctx, IndexMetadata{
		KnowledgeBase:     knowledgeBase,
		EmbeddingProvider: i.config.Provider,
		EmbeddingModel:    i.config.Model,
		EmbeddingDim:      i.config.Dim,
		BuiltAt:           i.now(),
	})
}

func (i *Indexer) BuildIndexFromSource(ctx context.Context, knowledgeBase string, source ChunkSource) error {
	chunks, err := source.Load(ctx)
	if err != nil {
		return err
	}
	return i.BuildIndex(ctx, knowledgeBase, chunks)
}

func buildEmbeddingText(chunk SearchChunk) string {
	lines := []string{
		"title: " + strings.TrimSpace(chunk.Title),
		"aliases: " + strings.Join(chunk.Aliases, ", "),
		"tags: " + strings.Join(chunk.Tags, ", "),
		"content: " + strings.TrimSpace(chunk.Content),
	}
	return strings.Join(lines, "\n")
}

func buildIndexedChunks(chunks []SearchChunk, vectors [][]float32, now time.Time) ([]IndexedChunk, error) {
	if len(chunks) != len(vectors) {
		return nil, ErrInvalidIndexBuildInput
	}
	indexed := make([]IndexedChunk, 0, len(chunks))
	for idx, chunk := range chunks {
		if chunk.ChunkID == "" {
			return nil, fmt.Errorf("%w: missing chunk id", ErrInvalidIndexBuildInput)
		}
		indexed = append(indexed, BuildIndexedChunk(chunk, vectors[idx], now))
	}
	return indexed, nil
}
