package search

import (
	"context"
	"errors"
)

// HybridSearcher 使用全文召回、向量召回和融合策略实现混合检索。
type HybridSearcher struct {
	knowledgeBase string
	store         HybridSearchStore
	embedder      Embedder
	fusion        FusionStrategy
	recallTopK    int
}

func NewHybridSearcher(
	knowledgeBase string,
	store HybridSearchStore,
	embedder Embedder,
	fusion FusionStrategy,
	recallTopK int,
) *HybridSearcher {
	if recallTopK <= 0 {
		recallTopK = defaultTopK
	}
	if fusion == nil {
		fusion = NewRRFFusion(defaultRRFK)
	}
	return &HybridSearcher{
		knowledgeBase: knowledgeBase,
		store:         store,
		embedder:      embedder,
		fusion:        fusion,
		recallTopK:    recallTopK,
	}
}

func (s *HybridSearcher) Search(ctx context.Context, request SearchRequest) ([]SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = normalizeSearchRequest(request)
	if err := validateSearchRequest(request); err != nil {
		return nil, err
	}

	ftsResults, ftsErr := s.store.SearchFTS(ctx, HybridSearchRequest{
		KnowledgeBase: s.knowledgeBase,
		Query:         request.Query,
		TopK:          maxInt(request.TopK, s.recallTopK),
	})

	vectorResults, vectorErr := s.searchVector(ctx, request)

	switch {
	case ftsErr == nil && vectorErr == nil:
		return s.fusion.Fuse(ftsResults, vectorResults, request.TopK), nil
	case ftsErr == nil:
		return candidatesToResults(ftsResults, request.TopK), nil
	case vectorErr == nil:
		return candidatesToResults(vectorResults, request.TopK), nil
	default:
		return nil, errors.Join(ftsErr, vectorErr)
	}
}

func (s *HybridSearcher) searchVector(ctx context.Context, request SearchRequest) ([]ScoredCandidate, error) {
	queryVector, err := EmbedQuery(ctx, s.embedder, request.Query)
	if err != nil {
		return nil, err
	}

	return s.store.SearchVector(ctx, VectorSearchRequest{
		KnowledgeBase: s.knowledgeBase,
		Query:         request.Query,
		QueryVector:   queryVector,
		TopK:          maxInt(request.TopK, s.recallTopK),
	})
}

func candidatesToResults(candidates []ScoredCandidate, topK int) []SearchResult {
	if topK <= 0 {
		topK = defaultTopK
	}
	results := make([]SearchResult, 0, len(candidates))
	for _, candidate := range candidates {
		score := candidate.VectorScore
		if candidate.FTSScore > score {
			score = candidate.FTSScore
		}
		results = append(results, SearchResult{
			ChunkID:       candidate.ChunkID,
			DocumentID:    candidate.DocumentID,
			KnowledgeBase: candidate.KnowledgeBase,
			SourceType:    candidate.SourceType,
			DocType:       candidate.DocType,
			Title:         candidate.Title,
			Content:       candidate.Content,
			SectionPath:   append([]string(nil), candidate.SectionPath...),
			Tags:          append([]string(nil), candidate.Tags...),
			Aliases:       append([]string(nil), candidate.Aliases...),
			Position:      candidate.Position,
			ChunkStrategy: candidate.ChunkStrategy,
			Score:         score,
		})
	}
	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
