package rag

import (
	"context"
	"errors"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

var ErrMissingGoldsetEntry = errors.New("missing approved goldset entry")

type Evaluator struct {
	lexical SearcherSet
	hybrid  SearcherSet
	ks      []int
	topK    int
}

func NewEvaluator(lexical SearcherSet, hybrid SearcherSet, ks []int, topK int) *Evaluator {
	if topK <= 0 {
		topK = 10
	}
	return &Evaluator{
		lexical: lexical,
		hybrid:  hybrid,
		ks:      append([]int(nil), ks...),
		topK:    topK,
	}
}

func (e *Evaluator) Evaluate(ctx context.Context, queries []Query, goldset []GoldsetEntry) (EvalReport, error) {
	goldByQueryID := make(map[string]GoldsetEntry, len(goldset))
	for _, entry := range goldset {
		goldByQueryID[entry.QueryID] = entry
	}

	records := make([]QueryEvalRecord, 0, len(queries)*2)
	for _, query := range queries {
		gold, ok := goldByQueryID[query.ID]
		if !ok {
			return EvalReport{}, ErrMissingGoldsetEntry
		}

		lexicalResults, err := e.search(ctx, e.lexical, query)
		if err != nil {
			return EvalReport{}, err
		}
		hybridResults, err := e.search(ctx, e.hybrid, query)
		if err != nil {
			return EvalReport{}, err
		}

		records = append(records,
			BuildQueryMetrics(query, gold, chunkIDsFromResults(lexicalResults), "lexical", e.ks),
			BuildQueryMetrics(query, gold, chunkIDsFromResults(hybridResults), "hybrid", e.ks),
		)
	}

	return EvalReport{
		QueryCount: len(queries),
		Metrics:    BuildMetricsReport(e.ks, records),
	}, nil
}

func (e *Evaluator) search(ctx context.Context, searchers SearcherSet, query Query) ([]retrievalsearch.SearchResult, error) {
	searcher, err := searchers.ForKnowledgeBase(query.KnowledgeBase)
	if err != nil {
		return nil, err
	}
	return searcher.Search(ctx, searchRequestForQuery(query, e.topK))
}

func chunkIDsFromResults(results []retrievalsearch.SearchResult) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.ChunkID)
	}
	return ids
}

func searchRequestForQuery(query Query, topK int) retrievalsearch.SearchRequest {
	return retrievalsearch.SearchRequest{
		Query: query.Query,
		TopK:  topK,
	}
}
