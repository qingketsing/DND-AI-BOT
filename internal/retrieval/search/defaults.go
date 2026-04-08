package search

import "path/filepath"

const projectRoot = "/home/qingke/DND-AI-BOT"

func DefaultRulesChunksPath() string {
	return filepath.Join(projectRoot, "data", "chunks", "rules", "chunks.jsonl")
}

func DefaultLoreChunksPath() string {
	return filepath.Join(projectRoot, "data", "chunks", "lore", "chunks.jsonl")
}

func NewDefaultRuleSearcher() (Searcher, error) {
	return NewRuleSearcher(DefaultRulesChunksPath())
}

func NewDefaultLoreSearcher() (Searcher, error) {
	return NewLoreSearcher(DefaultLoreChunksPath())
}
