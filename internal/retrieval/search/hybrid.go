package search

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

// HybridSearcher 使用全文召回、向量召回和融合策略实现混合检索。
type HybridSearcher struct {
	knowledgeBase string
	store         HybridSearchStore
	embedder      Embedder
	fusion        FusionStrategy
	recallTopK    int
	metrics       *observability.Metrics
	logger        *slog.Logger
}

type HybridSearcherOption func(*HybridSearcher)

func WithHybridSearchMetrics(metrics *observability.Metrics) HybridSearcherOption {
	return func(searcher *HybridSearcher) {
		if metrics != nil {
			searcher.metrics = metrics
		}
	}
}

func WithHybridSearchLogger(logger *slog.Logger) HybridSearcherOption {
	return func(searcher *HybridSearcher) {
		if logger != nil {
			searcher.logger = logger
		}
	}
}

func NewHybridSearcher(
	knowledgeBase string,
	store HybridSearchStore,
	embedder Embedder,
	fusion FusionStrategy,
	recallTopK int,
	options ...HybridSearcherOption,
) *HybridSearcher {
	if recallTopK <= 0 {
		recallTopK = defaultTopK
	}
	if fusion == nil {
		fusion = NewRRFFusion(defaultRRFK)
	}
	searcher := &HybridSearcher{
		knowledgeBase: knowledgeBase,
		store:         store,
		embedder:      embedder,
		fusion:        fusion,
		recallTopK:    recallTopK,
	}
	for _, option := range options {
		if option != nil {
			option(searcher)
		}
	}
	return searcher
}

func (s *HybridSearcher) Search(ctx context.Context, request SearchRequest) ([]SearchResult, error) {
	startedAt := time.Now()
	status := "error"
	defer func() {
		s.recordPhase("total", status, startedAt)
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = normalizeSearchRequest(request)
	if err := validateSearchRequest(request); err != nil {
		return nil, err
	}

	ftsStartedAt := time.Now()
	ftsResults, ftsErr := s.store.SearchFTS(ctx, HybridSearchRequest{
		KnowledgeBase: s.knowledgeBase,
		Query:         request.Query,
		TopK:          maxInt(request.TopK, s.recallTopK),
	})
	s.recordPhase("fts", statusFromError(ftsErr), ftsStartedAt)

	vectorResults, vectorErr := s.searchVector(ctx, request)

	switch {
	case ftsErr == nil && vectorErr == nil:
		fusionStartedAt := time.Now()
		results := s.fusion.Fuse(ftsResults, vectorResults, request.TopK)
		s.recordPhase("fusion", "success", fusionStartedAt)
		status = "success"
		return results, nil
	case ftsErr == nil:
		s.recordPhase("fusion", "skipped", time.Now())
		status = "degraded"
		return candidatesToResults(ftsResults, request.TopK), nil
	case vectorErr == nil:
		s.recordPhase("fusion", "skipped", time.Now())
		status = "degraded"
		return candidatesToResults(vectorResults, request.TopK), nil
	default:
		s.recordPhase("fusion", "skipped", time.Now())
		return nil, errors.Join(ftsErr, vectorErr)
	}
}

func (s *HybridSearcher) searchVector(ctx context.Context, request SearchRequest) ([]ScoredCandidate, error) {
	embeddingStartedAt := time.Now()
	queryVector, err := EmbedQuery(ctx, s.embedder, request.Query)
	if err != nil {
		s.recordPhase("embedding", "error", embeddingStartedAt)
		s.recordPhase("vector", "skipped", time.Now())
		return nil, err
	}
	s.recordPhase("embedding", "success", embeddingStartedAt)

	vectorStartedAt := time.Now()
	results, err := s.store.SearchVector(ctx, VectorSearchRequest{
		KnowledgeBase: s.knowledgeBase,
		Query:         request.Query,
		QueryVector:   queryVector,
		TopK:          maxInt(request.TopK, s.recallTopK),
	})
	s.recordPhase("vector", statusFromError(err), vectorStartedAt)
	return results, err
}

func (s *HybridSearcher) recordPhase(phase string, status string, startedAt time.Time) {
	if s.metrics == nil {
		return
	}
	observability.ObserveDuration(s.metrics.RAGPhaseDuration, prometheus.Labels{
		"knowledge_base": s.knowledgeBase,
		"phase":          phase,
		"status":         status,
	}, startedAt)
}

func statusFromError(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
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
