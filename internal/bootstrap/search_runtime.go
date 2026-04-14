package bootstrap

import (
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

type SearchBackend string

const (
	SearchBackendLexical SearchBackend = "lexical"
	SearchBackendHybrid  SearchBackend = "hybrid"
)

const defaultEmbeddingTimeoutSeconds = 30
const defaultEmbeddingBatchSize = 32

var (
	ErrInvalidSearchBackend             = errors.New("invalid SEARCH_BACKEND")
	ErrInvalidEmbeddingDim              = errors.New("invalid EMBEDDING_DIM")
	ErrInvalidEmbeddingBatchSize        = errors.New("invalid EMBEDDING_BATCH_SIZE")
	ErrInvalidEmbeddingTimeout          = errors.New("invalid EMBEDDING_TIMEOUT_SECONDS")
	ErrInvalidEmbeddingConfig           = retrievalsearch.ErrInvalidEmbeddingConfig
	ErrMissingSearchRuntimeDependencies = errors.New("missing search runtime dependencies")
)

type SearchConfig struct {
	Backend   SearchBackend
	Embedding retrievalsearch.EmbeddingConfig
}

// SearchRuntimeDependencies 承载已装配好的规则与设定检索器。
type SearchRuntimeDependencies struct {
	RuleSearcher retrievalsearch.Searcher
	LoreSearcher retrievalsearch.Searcher
}

func LoadSearchConfigFromEnv() (SearchConfig, error) {
	backend, err := parseSearchBackend(os.Getenv("SEARCH_BACKEND"))
	if err != nil {
		return SearchConfig{}, err
	}

	config := SearchConfig{Backend: backend}
	if backend != SearchBackendHybrid {
		return config, nil
	}

	dim, err := parsePositiveInt(os.Getenv("EMBEDDING_DIM"), ErrInvalidEmbeddingDim)
	if err != nil {
		return SearchConfig{}, err
	}
	batchSize, err := parsePositiveIntDefault(os.Getenv("EMBEDDING_BATCH_SIZE"), defaultEmbeddingBatchSize, ErrInvalidEmbeddingBatchSize)
	if err != nil {
		return SearchConfig{}, err
	}
	timeoutSeconds, err := parsePositiveIntDefault(os.Getenv("EMBEDDING_TIMEOUT_SECONDS"), defaultEmbeddingTimeoutSeconds, ErrInvalidEmbeddingTimeout)
	if err != nil {
		return SearchConfig{}, err
	}

	config.Embedding = retrievalsearch.NormalizeEmbeddingConfig(retrievalsearch.EmbeddingConfig{
		Provider:  os.Getenv("EMBEDDING_PROVIDER"),
		Model:     os.Getenv("EMBEDDING_MODEL"),
		Dim:       dim,
		BatchSize: batchSize,
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
		BaseURL:   os.Getenv("EMBEDDING_BASE_URL"),
		APIKey:    os.Getenv("EMBEDDING_API_KEY"),
	})
	if err := retrievalsearch.ValidateEmbeddingConfig(config.Embedding); err != nil {
		return SearchConfig{}, ErrInvalidEmbeddingConfig
	}

	return config, nil
}

// BuildSearchRuntime 构建默认的规则检索器和设定检索器。
func BuildSearchRuntime() (*SearchRuntimeDependencies, error) {
	return BuildSearchRuntimeWithDeps(nil)
}

// BuildSearchRuntimeWithDeps 根据运行时依赖构建规则检索器和设定检索器。
func BuildSearchRuntimeWithDeps(deps *RuntimeDependencies) (*SearchRuntimeDependencies, error) {
	config, err := LoadSearchConfigFromEnv()
	if err != nil {
		return nil, err
	}

	var (
		ruleSearcher retrievalsearch.Searcher
		loreSearcher retrievalsearch.Searcher
	)
	switch config.Backend {
	case SearchBackendLexical:
		ruleSearcher, err = retrievalsearch.NewDefaultRuleSearcher()
		if err != nil {
			return nil, err
		}
		loreSearcher, err = retrievalsearch.NewDefaultLoreSearcher()
		if err != nil {
			return nil, err
		}
	case SearchBackendHybrid:
		ruleSearcher, err = buildHybridSearcher(deps, retrievalsearch.KnowledgeBaseRules, config)
		if err != nil {
			return nil, err
		}
		loreSearcher, err = buildHybridSearcher(deps, retrievalsearch.KnowledgeBaseLore, config)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidSearchBackend
	}

	return &SearchRuntimeDependencies{
		RuleSearcher: ruleSearcher,
		LoreSearcher: loreSearcher,
	}, nil
}

func parseSearchBackend(value string) (SearchBackend, error) {
	switch strings.TrimSpace(value) {
	case "", string(SearchBackendLexical):
		return SearchBackendLexical, nil
	case string(SearchBackendHybrid):
		return SearchBackendHybrid, nil
	default:
		return "", ErrInvalidSearchBackend
	}
}

func parsePositiveInt(value string, errValue error) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errValue
	}
	return parsed, nil
}

func parsePositiveIntDefault(value string, defaultValue int, errValue error) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue, nil
	}
	return parsePositiveInt(value, errValue)
}

func buildHybridSearcher(
	deps *RuntimeDependencies,
	knowledgeBase string,
	config SearchConfig,
) (retrievalsearch.Searcher, error) {
	store, err := buildHybridSearchStore(deps)
	if err != nil {
		return nil, err
	}
	embedder, err := buildEmbeddingProvider(config)
	if err != nil {
		return nil, err
	}

	return retrievalsearch.NewHybridSearcher(
		knowledgeBase,
		store,
		embedder,
		retrievalsearch.NewRRFFusion(60),
		20,
	), nil
}

func buildHybridSearchStore(deps *RuntimeDependencies) (*retrievalsearch.PostgresHybridSearchStore, error) {
	if deps == nil || deps.DB == nil {
		return nil, ErrMissingSearchRuntimeDependencies
	}
	return retrievalsearch.NewPostgresHybridSearchStore(deps.DB), nil
}

func buildEmbeddingProvider(config SearchConfig) (retrievalsearch.Embedder, error) {
	switch config.Embedding.Provider {
	case retrievalsearch.EmbeddingProviderQwen:
		return retrievalsearch.NewQwenEmbedder(config.Embedding)
	default:
		return nil, ErrInvalidEmbeddingConfig
	}
}

func newEmptySearchRuntimeDB() *sql.DB {
	return &sql.DB{}
}
