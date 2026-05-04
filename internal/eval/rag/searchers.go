package rag

import (
	"database/sql"
	"errors"

	"DND-AI-BOT/internal/bootstrap"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

var ErrHybridEvalRequiresHybridBackend = errors.New("rag eval hybrid search requires SEARCH_BACKEND=hybrid")

type SearcherSet struct {
	RuleSearcher retrievalsearch.Searcher
	LoreSearcher retrievalsearch.Searcher
}

func BuildLexicalSearchers() (SearcherSet, error) {
	ruleSearcher, err := retrievalsearch.NewDefaultRuleSearcher()
	if err != nil {
		return SearcherSet{}, err
	}
	loreSearcher, err := retrievalsearch.NewDefaultLoreSearcher()
	if err != nil {
		return SearcherSet{}, err
	}
	return SearcherSet{
		RuleSearcher: ruleSearcher,
		LoreSearcher: loreSearcher,
	}, nil
}

func BuildHybridSearchers(db *sql.DB) (SearcherSet, error) {
	config, err := bootstrap.LoadSearchConfigFromEnv()
	if err != nil {
		return SearcherSet{}, err
	}
	if config.Backend != bootstrap.SearchBackendHybrid {
		return SearcherSet{}, ErrHybridEvalRequiresHybridBackend
	}
	store := retrievalsearch.NewPostgresHybridSearchStore(db)
	embedder, err := retrievalsearch.NewQwenEmbedder(config.Embedding)
	if err != nil {
		return SearcherSet{}, err
	}
	return SearcherSet{
		RuleSearcher: retrievalsearch.NewHybridSearcher(
			retrievalsearch.KnowledgeBaseRules,
			store,
			embedder,
			retrievalsearch.NewRRFFusion(60),
			20,
		),
		LoreSearcher: retrievalsearch.NewHybridSearcher(
			retrievalsearch.KnowledgeBaseLore,
			store,
			embedder,
			retrievalsearch.NewRRFFusion(60),
			20,
		),
	}, nil
}

func MergeCandidatePools(resultSets ...[]retrievalsearch.SearchResult) []CandidateChunk {
	candidates := make([]CandidateChunk, 0)
	seen := make(map[string]struct{})
	for _, results := range resultSets {
		for _, result := range results {
			if _, ok := seen[result.ChunkID]; ok {
				continue
			}
			seen[result.ChunkID] = struct{}{}
			candidates = append(candidates, CandidateChunk{
				ChunkID: result.ChunkID,
				Title:   result.Title,
				Content: result.Content,
			})
		}
	}
	return candidates
}

func (s SearcherSet) ForKnowledgeBase(knowledgeBase string) (retrievalsearch.Searcher, error) {
	switch knowledgeBase {
	case retrievalsearch.KnowledgeBaseRules:
		if s.RuleSearcher == nil {
			return nil, errors.New("missing rules searcher")
		}
		return s.RuleSearcher, nil
	case retrievalsearch.KnowledgeBaseLore:
		if s.LoreSearcher == nil {
			return nil, errors.New("missing lore searcher")
		}
		return s.LoreSearcher, nil
	default:
		return nil, errors.New("unsupported knowledge base")
	}
}
