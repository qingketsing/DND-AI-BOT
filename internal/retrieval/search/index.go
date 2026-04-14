package search

import "time"

// IndexedChunk 表示已写入混合检索索引的知识块。
type IndexedChunk struct {
	ID            string
	KnowledgeBase string
	SourceID      string
	Title         string
	Aliases       []string
	Content       string
	Metadata      map[string]any
	Embedding     []float32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
