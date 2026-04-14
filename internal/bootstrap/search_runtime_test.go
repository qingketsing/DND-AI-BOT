package bootstrap

import (
	"testing"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestBuildSearchRuntimeBuildsDefaultSearchers(t *testing.T) {
	deps, err := BuildSearchRuntime()
	if err != nil {
		t.Fatalf("expected search runtime build to succeed, got %v", err)
	}
	if deps == nil {
		t.Fatal("expected search runtime dependencies to be created")
	}
	if deps.RuleSearcher == nil {
		t.Fatal("expected rule searcher to be created")
	}
	if deps.LoreSearcher == nil {
		t.Fatal("expected lore searcher to be created")
	}
}

func TestLoadSearchConfigFromEnvDefaultsToLexical(t *testing.T) {
	t.Setenv("SEARCH_BACKEND", "")
	t.Setenv("EMBEDDING_PROVIDER", "")
	t.Setenv("EMBEDDING_MODEL", "")
	t.Setenv("EMBEDDING_API_KEY", "")
	t.Setenv("EMBEDDING_BASE_URL", "")
	t.Setenv("EMBEDDING_DIM", "")
	t.Setenv("EMBEDDING_BATCH_SIZE", "")
	t.Setenv("EMBEDDING_TIMEOUT_SECONDS", "")

	config, err := LoadSearchConfigFromEnv()
	if err != nil {
		t.Fatalf("expected lexical default config, got %v", err)
	}
	if config.Backend != SearchBackendLexical {
		t.Fatalf("expected lexical backend, got %q", config.Backend)
	}
}

func TestLoadSearchConfigFromEnvRejectsHybridWithoutEmbeddingCredentials(t *testing.T) {
	t.Setenv("SEARCH_BACKEND", "hybrid")
	t.Setenv("EMBEDDING_PROVIDER", "qwen")
	t.Setenv("EMBEDDING_MODEL", "qwen-embed-8b")
	t.Setenv("EMBEDDING_API_KEY", "")
	t.Setenv("EMBEDDING_BASE_URL", "")
	t.Setenv("EMBEDDING_DIM", "1024")
	t.Setenv("EMBEDDING_BATCH_SIZE", "32")
	t.Setenv("EMBEDDING_TIMEOUT_SECONDS", "30")

	_, err := LoadSearchConfigFromEnv()
	if err != ErrInvalidEmbeddingConfig {
		t.Fatalf("expected ErrInvalidEmbeddingConfig, got %v", err)
	}
}

func TestBuildSearchRuntimeWithDepsRejectsHybridWithoutDB(t *testing.T) {
	t.Setenv("SEARCH_BACKEND", "hybrid")
	t.Setenv("EMBEDDING_PROVIDER", "qwen")
	t.Setenv("EMBEDDING_MODEL", "qwen-embed-8b")
	t.Setenv("EMBEDDING_API_KEY", "secret")
	t.Setenv("EMBEDDING_BASE_URL", "https://example.com")
	t.Setenv("EMBEDDING_DIM", "1024")
	t.Setenv("EMBEDDING_BATCH_SIZE", "32")
	t.Setenv("EMBEDDING_TIMEOUT_SECONDS", "30")

	_, err := BuildSearchRuntimeWithDeps(nil)
	if err != ErrMissingSearchRuntimeDependencies {
		t.Fatalf("expected ErrMissingSearchRuntimeDependencies, got %v", err)
	}
}

func TestBuildSearchRuntimeWithDepsBuildsHybridSearchers(t *testing.T) {
	t.Setenv("SEARCH_BACKEND", "hybrid")
	t.Setenv("EMBEDDING_PROVIDER", "qwen")
	t.Setenv("EMBEDDING_MODEL", "qwen-embed-8b")
	t.Setenv("EMBEDDING_API_KEY", "secret")
	t.Setenv("EMBEDDING_BASE_URL", "https://example.com")
	t.Setenv("EMBEDDING_DIM", "1024")
	t.Setenv("EMBEDDING_BATCH_SIZE", "32")
	t.Setenv("EMBEDDING_TIMEOUT_SECONDS", "30")

	deps, err := BuildSearchRuntimeWithDeps(&RuntimeDependencies{DB: newEmptySearchRuntimeDB()})
	if err != nil {
		t.Fatalf("expected hybrid search runtime to build, got %v", err)
	}
	if deps.RuleSearcher == nil || deps.LoreSearcher == nil {
		t.Fatal("expected hybrid searchers to be constructed")
	}
	if _, ok := deps.RuleSearcher.(*retrievalsearch.HybridSearcher); !ok {
		t.Fatalf("expected rule searcher to be HybridSearcher, got %T", deps.RuleSearcher)
	}
}
