package bootstrap

import retrievalsearch "DND-AI-BOT/internal/retrieval/search"

// SearchRuntimeDependencies 承载已装配好的规则与设定检索器。
type SearchRuntimeDependencies struct {
	RuleSearcher retrievalsearch.Searcher
	LoreSearcher retrievalsearch.Searcher
}

// BuildSearchRuntime 构建默认的规则检索器和设定检索器。
func BuildSearchRuntime() (*SearchRuntimeDependencies, error) {
	ruleSearcher, err := retrievalsearch.NewDefaultRuleSearcher()
	if err != nil {
		return nil, err
	}
	loreSearcher, err := retrievalsearch.NewDefaultLoreSearcher()
	if err != nil {
		return nil, err
	}

	return &SearchRuntimeDependencies{
		RuleSearcher: ruleSearcher,
		LoreSearcher: loreSearcher,
	}, nil
}
