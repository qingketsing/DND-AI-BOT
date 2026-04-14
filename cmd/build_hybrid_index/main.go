package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"DND-AI-BOT/internal/bootstrap"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

var ErrInvalidKnowledgeBase = errors.New("invalid knowledge base")

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Fatalf("build hybrid index failed: %v", err)
	}
}

func run(ctx context.Context, args []string) error {
	projectRoot, err := resolveProjectRoot()
	if err != nil {
		return err
	}
	_ = loadDotEnv(filepath.Join(projectRoot, ".env"))

	knowledgeBase, err := parseKnowledgeBaseArg(args)
	if err != nil {
		return err
	}

	embeddingConfig, err := loadEmbeddingConfigFromEnv()
	if err != nil {
		return err
	}

	db, err := bootstrap.OpenPostgresFromEnv()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := bootstrap.RunEmbeddedMigrations(ctx, db); err != nil {
		return err
	}

	store := retrievalsearch.NewPostgresHybridSearchStore(db)
	metadataStore := retrievalsearch.NewPostgresIndexMetadataStore(db)
	embedder, err := retrievalsearch.NewQwenEmbedder(embeddingConfig)
	if err != nil {
		return err
	}
	indexer := retrievalsearch.NewIndexer(store, metadataStore, embedder, embeddingConfig)

	for _, base := range expandKnowledgeBases(knowledgeBase) {
		chunks, err := loadChunks(base)
		if err != nil {
			return err
		}
		log.Printf("building hybrid index: knowledge_base=%s chunk_count=%d", base, len(chunks))
		if err := indexer.BuildIndex(ctx, base, chunks); err != nil {
			return err
		}
		log.Printf("hybrid index build finished: knowledge_base=%s chunk_count=%d", base, len(chunks))
	}

	return nil
}

func parseKnowledgeBaseArg(args []string) (string, error) {
	fs := flag.NewFlagSet("build_hybrid_index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var knowledgeBase string
	fs.StringVar(&knowledgeBase, "knowledge-base", "", "knowledge base to build: rules|lore|all")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	switch knowledgeBase {
	case retrievalsearch.KnowledgeBaseRules, retrievalsearch.KnowledgeBaseLore, "all":
		return knowledgeBase, nil
	default:
		return "", ErrInvalidKnowledgeBase
	}
}

func expandKnowledgeBases(knowledgeBase string) []string {
	if knowledgeBase == "all" {
		return []string{retrievalsearch.KnowledgeBaseRules, retrievalsearch.KnowledgeBaseLore}
	}
	return []string{knowledgeBase}
}

func loadChunks(knowledgeBase string) ([]retrievalsearch.SearchChunk, error) {
	switch knowledgeBase {
	case retrievalsearch.KnowledgeBaseRules:
		return retrievalsearch.LoadChunksFromJSONL(retrievalsearch.DefaultRulesChunksPath())
	case retrievalsearch.KnowledgeBaseLore:
		return retrievalsearch.LoadChunksFromJSONL(retrievalsearch.DefaultLoreChunksPath())
	default:
		return nil, ErrInvalidKnowledgeBase
	}
}

func loadEmbeddingConfigFromEnv() (retrievalsearch.EmbeddingConfig, error) {
	config := retrievalsearch.NormalizeEmbeddingConfig(retrievalsearch.EmbeddingConfig{
		Provider:  os.Getenv("EMBEDDING_PROVIDER"),
		Model:     os.Getenv("EMBEDDING_MODEL"),
		BaseURL:   os.Getenv("EMBEDDING_BASE_URL"),
		APIKey:    os.Getenv("EMBEDDING_API_KEY"),
		Dim:       mustReadPositiveInt("EMBEDDING_DIM"),
		BatchSize: mustReadPositiveIntDefault("EMBEDDING_BATCH_SIZE", 32),
		Timeout:   time.Duration(mustReadPositiveIntDefault("EMBEDDING_TIMEOUT_SECONDS", 30)) * time.Second,
	})
	if err := retrievalsearch.ValidateEmbeddingConfig(config); err != nil {
		return retrievalsearch.EmbeddingConfig{}, err
	}
	return config, nil
}

func resolveProjectRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	start := current
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start, nil
		}
		current = parent
	}
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func mustReadPositiveInt(envName string) int {
	value := os.Getenv(envName)
	if value == "" {
		return 0
	}
	var parsed int
	_, _ = fmt.Sscanf(value, "%d", &parsed)
	return parsed
}

func mustReadPositiveIntDefault(envName string, fallback int) int {
	value := mustReadPositiveInt(envName)
	if value <= 0 {
		return fallback
	}
	return value
}
