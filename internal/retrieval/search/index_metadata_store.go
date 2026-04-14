package search

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrIndexMetadataNotFound = errors.New("index metadata not found")

type IndexMetadata struct {
	KnowledgeBase     string
	EmbeddingProvider string
	EmbeddingModel    string
	EmbeddingDim      int
	BuiltAt           time.Time
}

type IndexMetadataStore interface {
	UpsertIndexMetadata(ctx context.Context, metadata IndexMetadata) error
	LoadIndexMetadata(ctx context.Context, knowledgeBase string) (*IndexMetadata, error)
}

type PostgresIndexMetadataStore struct {
	db *sql.DB
}

func NewPostgresIndexMetadataStore(db *sql.DB) *PostgresIndexMetadataStore {
	return &PostgresIndexMetadataStore{db: db}
}

func (s *PostgresIndexMetadataStore) UpsertIndexMetadata(ctx context.Context, metadata IndexMetadata) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO knowledge_index_metadata (
			knowledge_base,
			embedding_provider,
			embedding_model,
			embedding_dim,
			built_at
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (knowledge_base) DO UPDATE
		SET embedding_provider = EXCLUDED.embedding_provider,
		    embedding_model = EXCLUDED.embedding_model,
		    embedding_dim = EXCLUDED.embedding_dim,
		    built_at = EXCLUDED.built_at
	`,
		metadata.KnowledgeBase,
		metadata.EmbeddingProvider,
		metadata.EmbeddingModel,
		metadata.EmbeddingDim,
		metadata.BuiltAt,
	)
	return err
}

func (s *PostgresIndexMetadataStore) LoadIndexMetadata(ctx context.Context, knowledgeBase string) (*IndexMetadata, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT knowledge_base, embedding_provider, embedding_model, embedding_dim, built_at
		FROM knowledge_index_metadata
		WHERE knowledge_base = $1
	`, knowledgeBase)

	var metadata IndexMetadata
	if err := row.Scan(
		&metadata.KnowledgeBase,
		&metadata.EmbeddingProvider,
		&metadata.EmbeddingModel,
		&metadata.EmbeddingDim,
		&metadata.BuiltAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIndexMetadataNotFound
		}
		return nil, err
	}

	return &metadata, nil
}
