package search

import "context"

// HybridSearchRequest 表示全文检索召回请求。
type HybridSearchRequest struct {
	KnowledgeBase string
	Query         string
	TopK          int
}

// VectorSearchRequest 表示向量召回请求。
type VectorSearchRequest struct {
	KnowledgeBase string
	Query         string
	QueryVector   []float32
	TopK          int
}

// ScoredCandidate 表示召回阶段的候选块。
type ScoredCandidate struct {
	ChunkID       string
	DocumentID    string
	KnowledgeBase string
	SourceType    string
	DocType       string
	Title         string
	Content       string
	SectionPath   []string
	Tags          []string
	Aliases       []string
	Position      int
	ChunkStrategy string
	FTSScore      float64
	VectorScore   float64
}

// HybridSearchStore 定义混合检索后端需要的全文与向量召回接口。
type HybridSearchStore interface {
	SearchFTS(ctx context.Context, request HybridSearchRequest) ([]ScoredCandidate, error)
	SearchVector(ctx context.Context, request VectorSearchRequest) ([]ScoredCandidate, error)
}
