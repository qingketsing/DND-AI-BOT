package tools

import (
	"context"
	"encoding/json"

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
}

type SearchRulesTool struct {
	searcher retrievalsearch.Searcher
}

func NewSearchRulesTool(searcher retrievalsearch.Searcher) *SearchRulesTool {
	return &SearchRulesTool{searcher: searcher}
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
	return callKnowledgeSearcher(ctx, t.Spec().Name, retrievalsearch.KnowledgeBaseRules, t.searcher, input.Raw)
}

type SearchLoreTool struct {
	searcher retrievalsearch.Searcher
}

func NewSearchLoreTool(searcher retrievalsearch.Searcher) *SearchLoreTool {
	return &SearchLoreTool{searcher: searcher}
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
	return callKnowledgeSearcher(ctx, t.Spec().Name, retrievalsearch.KnowledgeBaseLore, t.searcher, input.Raw)
}

func callKnowledgeSearcher(
	ctx context.Context,
	toolName string,
	knowledgeBase string,
	searcher retrievalsearch.Searcher,
	raw json.RawMessage,
) (CallOutput, error) {
	var args searchKnowledgeArgs
	if err := decodeToolInput(raw, &args); err != nil {
		return CallOutput{}, err
	}
	if args.TopK <= 0 {
		args.TopK = defaultKnowledgeSearchTopK
	}

	results, err := searcher.Search(ctx, retrievalsearch.SearchRequest{
		Query: args.Query,
		TopK:  args.TopK,
	})
	if err != nil {
		return CallOutput{}, err
	}

	return newToolOutput(toolName, searchKnowledgeResult{
		KnowledgeBase: knowledgeBase,
		Results:       results,
	}), nil
}
