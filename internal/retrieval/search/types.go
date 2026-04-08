package search

import (
	"context"
	"errors"
)

const (
	KnowledgeBaseRules = "rules"
	KnowledgeBaseLore  = "lore"
)

var ErrInvalidSearchRequest = errors.New("invalid search request")

type SearchChunk struct {
	ChunkID       string   `json:"chunk_id"`
	DocumentID    string   `json:"document_id"`
	KnowledgeBase string   `json:"knowledge_base"`
	SourceType    string   `json:"source_type"`
	DocType       string   `json:"doc_type"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Language      string   `json:"language"`
	SectionPath   []string `json:"section_path"`
	Tags          []string `json:"tags"`
	Aliases       []string `json:"aliases"`
	Position      int      `json:"position"`
	ChunkStrategy string   `json:"chunk_strategy"`
}

type SearchResult struct {
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
	Score         float64
}

type SearchRequest struct {
	Query string
	TopK  int
}

type Searcher interface {
	Search(ctx context.Context, request SearchRequest) ([]SearchResult, error)
}
