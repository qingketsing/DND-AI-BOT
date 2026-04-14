package bootstrap

import "testing"

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

func TestBuildSearchRuntimeRejectsHybridUntilSearcherIsImplemented(t *testing.T) {
	t.Setenv("SEARCH_BACKEND", "hybrid")
	t.Setenv("EMBEDDING_PROVIDER", "qwen")
	t.Setenv("EMBEDDING_MODEL", "qwen-embed-8b")
	t.Setenv("EMBEDDING_API_KEY", "secret")
	t.Setenv("EMBEDDING_BASE_URL", "https://example.com")
	t.Setenv("EMBEDDING_DIM", "1024")
	t.Setenv("EMBEDDING_BATCH_SIZE", "32")
	t.Setenv("EMBEDDING_TIMEOUT_SECONDS", "30")

	_, err := BuildSearchRuntime()
	if err != ErrHybridSearchNotImplemented {
		t.Fatalf("expected ErrHybridSearchNotImplemented, got %v", err)
	}
}
