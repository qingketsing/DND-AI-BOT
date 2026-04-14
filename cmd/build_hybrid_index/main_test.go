package main

import (
	"os"
	"path/filepath"
	"testing"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestParseKnowledgeBaseArg(t *testing.T) {
	t.Parallel()

	got, err := parseKnowledgeBaseArg([]string{"--knowledge-base", "rules"})
	if err != nil {
		t.Fatalf("parseKnowledgeBaseArg() error = %v", err)
	}
	if got != retrievalsearch.KnowledgeBaseRules {
		t.Fatalf("got %q, want %q", got, retrievalsearch.KnowledgeBaseRules)
	}
}

func TestLoadDotEnvSetsEnvValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("EMBEDDING_MODEL=Qwen/Qwen3-Embedding-8B\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	original, hadOriginal := os.LookupEnv("EMBEDDING_MODEL")
	if err := os.Unsetenv("EMBEDDING_MODEL"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	t.Cleanup(func() {
		if hadOriginal {
			_ = os.Setenv("EMBEDDING_MODEL", original)
			return
		}
		_ = os.Unsetenv("EMBEDDING_MODEL")
	})

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if got := os.Getenv("EMBEDDING_MODEL"); got != "Qwen/Qwen3-Embedding-8B" {
		t.Fatalf("EMBEDDING_MODEL = %q", got)
	}
}
