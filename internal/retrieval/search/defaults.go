package search

import (
	"os"
	"path/filepath"
)

const projectRootEnv = "DND_AI_BOT_ROOT"

func DefaultRulesChunksPath() string {
	root := resolveProjectRootFromWorkingDir()
	return buildChunksPath(root, KnowledgeBaseRules)
}

func DefaultLoreChunksPath() string {
	root := resolveProjectRootFromWorkingDir()
	return buildChunksPath(root, KnowledgeBaseLore)
}

func NewDefaultRuleSearcher() (Searcher, error) {
	return NewRuleSearcher(DefaultRulesChunksPath())
}

func NewDefaultLoreSearcher() (Searcher, error) {
	return NewLoreSearcher(DefaultLoreChunksPath())
}

func resolveProjectRootFromWorkingDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return "."
	}

	return resolveProjectRoot(workingDir)
}

func resolveProjectRoot(start string) string {
	if configuredRoot := os.Getenv(projectRootEnv); configuredRoot != "" {
		return configuredRoot
	}

	current := start
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return start
		}
		current = parent
	}
}

func buildChunksPath(projectRoot string, knowledgeBase string) string {
	return filepath.Join(projectRoot, "data", "chunks", knowledgeBase, "chunks.jsonl")
}
