package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

const defaultKnowledgeSearchTopK = 5

type searchKnowledgeArgs struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

type searchKnowledgeResult struct {
	KnowledgeBase string                         `json:"knowledge_base"`
	Results       []retrievalsearch.SearchResult `json:"results"`
	Degraded      bool                           `json:"degraded"`
	Message       string                         `json:"message,omitempty"`
}

type searchToolOptions struct {
	metrics *observability.Metrics
}

// SearchToolOption 定义知识库检索工具可选配置。
type SearchToolOption func(*searchToolOptions)

// WithSearchToolMetrics 注入知识库检索指标。
func WithSearchToolMetrics(metrics *observability.Metrics) SearchToolOption {
	return func(options *searchToolOptions) {
		if metrics != nil {
			options.metrics = metrics
		}
	}
}

type SearchRulesTool struct {
	searcher retrievalsearch.Searcher
	metrics  *observability.Metrics
}

func NewSearchRulesTool(searcher retrievalsearch.Searcher, options ...SearchToolOption) *SearchRulesTool {
	opts := buildSearchToolOptions(options...)
	return &SearchRulesTool{
		searcher: searcher,
		metrics:  opts.metrics,
	}
}

func (t *SearchRulesTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "search_rules",
		Description: "检索规则知识库，返回与查询最相关的规则片段。",
		InputSchema: objectSchema(map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "要检索的规则问题或关键词。",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "返回结果数量，默认 5。",
			},
		}, "query"),
	}
}

func (t *SearchRulesTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	return callKnowledgeSearcher(ctx, t.Spec().Name, retrievalsearch.KnowledgeBaseRules, t.searcher, t.metrics, input.Raw)
}

type SearchLoreTool struct {
	searcher retrievalsearch.Searcher
	metrics  *observability.Metrics
}

func NewSearchLoreTool(searcher retrievalsearch.Searcher, options ...SearchToolOption) *SearchLoreTool {
	opts := buildSearchToolOptions(options...)
	return &SearchLoreTool{
		searcher: searcher,
		metrics:  opts.metrics,
	}
}

func (t *SearchLoreTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "search_lore",
		Description: "检索世界观与设定知识库，返回与查询最相关的设定片段。",
		InputSchema: objectSchema(map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "要检索的设定问题或关键词。",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "返回结果数量，默认 5。",
			},
		}, "query"),
	}
}

func (t *SearchLoreTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	return callKnowledgeSearcher(ctx, t.Spec().Name, retrievalsearch.KnowledgeBaseLore, t.searcher, t.metrics, input.Raw)
}

func callKnowledgeSearcher(
	ctx context.Context,
	toolName string,
	knowledgeBase string,
	searcher retrievalsearch.Searcher,
	metrics *observability.Metrics,
	raw json.RawMessage,
) (CallOutput, error) {
	var args searchKnowledgeArgs
	if err := decodeToolInput(raw, &args); err != nil {
		return CallOutput{}, err
	}
	if args.TopK <= 0 {
		args.TopK = defaultKnowledgeSearchTopK
	}

	startedAt := time.Now()
	results, err := searcher.Search(ctx, retrievalsearch.SearchRequest{
		Query: args.Query,
		TopK:  args.TopK,
	})
	if err != nil {
		recordRAGSearch(metrics, knowledgeBase, "degraded", startedAt)
		recordRAGDegraded(metrics, knowledgeBase)
		return newToolOutput(toolName, buildRetrievalFallbackResult(args.Query, knowledgeBase, err)), nil
	}

	recordRAGSearch(metrics, knowledgeBase, "success", startedAt)
	return newToolOutput(toolName, searchKnowledgeResult{
		KnowledgeBase: knowledgeBase,
		Results:       results,
	}), nil
}

func buildSearchToolOptions(options ...SearchToolOption) searchToolOptions {
	var opts searchToolOptions
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	return opts
}

func recordRAGSearch(metrics *observability.Metrics, knowledgeBase string, status string, startedAt time.Time) {
	if metrics == nil {
		return
	}
	labels := prometheus.Labels{
		"knowledge_base": knowledgeBase,
		"status":         status,
	}
	metrics.RAGSearchTotal.With(labels).Inc()
	observability.ObserveDuration(metrics.RAGSearchDuration, labels, startedAt)
}

func recordRAGDegraded(metrics *observability.Metrics, knowledgeBase string) {
	if metrics == nil {
		return
	}
	metrics.RAGDegradedTotal.With(prometheus.Labels{"knowledge_base": knowledgeBase}).Inc()
}

func buildRetrievalFallbackResult(query string, knowledgeBase string, err error) searchKnowledgeResult {
	_ = query
	_ = err
	return searchKnowledgeResult{
		KnowledgeBase: knowledgeBase,
		Results:       []retrievalsearch.SearchResult{},
		Degraded:      true,
		Message:       "知识库检索暂时不可用，请基于已确认上下文继续；涉及具体规则或设定细节时，需要稍后重新检索确认。",
	}
}
