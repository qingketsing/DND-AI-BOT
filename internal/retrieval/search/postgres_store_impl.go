package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PostgresHybridSearchStore 基于 PostgreSQL 提供全文与向量召回。
type PostgresHybridSearchStore struct {
	db *sql.DB
}

func NewPostgresHybridSearchStore(db *sql.DB) *PostgresHybridSearchStore {
	return &PostgresHybridSearchStore{db: db}
}

func (s *PostgresHybridSearchStore) UpsertChunks(ctx context.Context, chunks []IndexedChunk) error {
	for _, chunk := range chunks {
		metadata, err := json.Marshal(chunk.Metadata)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO knowledge_chunks (
				id,
				knowledge_base,
				source_id,
				title,
				aliases,
				content,
				metadata,
				content_tsv,
				embedding,
				created_at,
				updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5::text[], $6, $7,
				to_tsvector('simple', concat_ws(' ', $4, array_to_string($5::text[], ' '), $6)),
				$8::vector, $9, $10
			)
			ON CONFLICT (id) DO UPDATE
			SET knowledge_base = EXCLUDED.knowledge_base,
			    source_id = EXCLUDED.source_id,
			    title = EXCLUDED.title,
			    aliases = EXCLUDED.aliases,
			    content = EXCLUDED.content,
			    metadata = EXCLUDED.metadata,
			    content_tsv = EXCLUDED.content_tsv,
			    embedding = EXCLUDED.embedding,
			    updated_at = EXCLUDED.updated_at
		`,
			chunk.ID,
			chunk.KnowledgeBase,
			chunk.SourceID,
			chunk.Title,
			formatTextArrayLiteral(chunk.Aliases),
			chunk.Content,
			metadata,
			formatVectorLiteral(chunk.Embedding),
			chunk.CreatedAt,
			chunk.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresHybridSearchStore) SearchFTS(ctx context.Context, request HybridSearchRequest) ([]ScoredCandidate, error) {
	request = normalizeHybridSearchRequest(request)
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			source_id,
			knowledge_base,
			title,
			content,
			metadata,
			ts_rank(
				content_tsv,
				websearch_to_tsquery('simple', $2)
			) AS score
		FROM knowledge_chunks
		WHERE knowledge_base = $1
		  AND content_tsv @@ websearch_to_tsquery('simple', $2)
		ORDER BY score DESC, id ASC
		LIMIT $3
	`, request.KnowledgeBase, request.Query, request.TopK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanKnowledgeChunkCandidates(rows, func(candidate *ScoredCandidate, score float64) {
		candidate.FTSScore = score
	})
}

func (s *PostgresHybridSearchStore) SearchVector(ctx context.Context, request VectorSearchRequest) ([]ScoredCandidate, error) {
	request = normalizeVectorSearchRequest(request)
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			source_id,
			knowledge_base,
			title,
			content,
			metadata,
			1 - (embedding <=> $2::vector) AS score
		FROM knowledge_chunks
		WHERE knowledge_base = $1
		ORDER BY embedding <=> $2::vector ASC, id ASC
		LIMIT $3
	`, request.KnowledgeBase, formatVectorLiteral(request.QueryVector), request.TopK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanKnowledgeChunkCandidates(rows, func(candidate *ScoredCandidate, score float64) {
		candidate.VectorScore = score
	})
}

func scanKnowledgeChunkCandidates(rows *sql.Rows, assignScore func(candidate *ScoredCandidate, score float64)) ([]ScoredCandidate, error) {
	results := make([]ScoredCandidate, 0)
	for rows.Next() {
		var (
			candidate ScoredCandidate
			rawMeta   []byte
			score     float64
		)
		if err := rows.Scan(
			&candidate.ChunkID,
			&candidate.DocumentID,
			&candidate.KnowledgeBase,
			&candidate.Title,
			&candidate.Content,
			&rawMeta,
			&score,
		); err != nil {
			return nil, err
		}
		if err := hydrateCandidateMetadata(&candidate, rawMeta); err != nil {
			return nil, err
		}
		assignScore(&candidate, score)
		results = append(results, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func hydrateCandidateMetadata(candidate *ScoredCandidate, rawMeta []byte) error {
	if len(rawMeta) == 0 {
		return nil
	}
	var meta struct {
		SourceType    string   `json:"source_type"`
		DocType       string   `json:"doc_type"`
		SectionPath   []string `json:"section_path"`
		Tags          []string `json:"tags"`
		Aliases       []string `json:"aliases"`
		Position      int      `json:"position"`
		ChunkStrategy string   `json:"chunk_strategy"`
	}
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		return err
	}
	candidate.SourceType = meta.SourceType
	candidate.DocType = meta.DocType
	candidate.SectionPath = meta.SectionPath
	candidate.Tags = meta.Tags
	candidate.Aliases = meta.Aliases
	candidate.Position = meta.Position
	candidate.ChunkStrategy = meta.ChunkStrategy
	return nil
}

func normalizeHybridSearchRequest(request HybridSearchRequest) HybridSearchRequest {
	request.Query = strings.TrimSpace(request.Query)
	if request.TopK <= 0 {
		request.TopK = defaultTopK
	}
	return request
}

func normalizeVectorSearchRequest(request VectorSearchRequest) VectorSearchRequest {
	request.Query = strings.TrimSpace(request.Query)
	if request.TopK <= 0 {
		request.TopK = defaultTopK
	}
	return request
}

func formatVectorLiteral(vector []float32) string {
	parts := make([]string, 0, len(vector))
	for _, value := range vector {
		parts = append(parts, fmt.Sprintf("%g", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func formatTextArrayLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ReplaceAll(value, `\`, `\\`)
		value = strings.ReplaceAll(value, `"`, `\"`)
		escaped = append(escaped, `"`+value+`"`)
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

func BuildIndexedChunk(chunk SearchChunk, embedding []float32, now time.Time) IndexedChunk {
	return IndexedChunk{
		ID:            chunk.ChunkID,
		KnowledgeBase: chunk.KnowledgeBase,
		SourceID:      chunk.DocumentID,
		Title:         chunk.Title,
		Aliases:       append([]string(nil), chunk.Aliases...),
		Content:       chunk.Content,
		Metadata: map[string]any{
			"source_type":    chunk.SourceType,
			"doc_type":       chunk.DocType,
			"section_path":   append([]string(nil), chunk.SectionPath...),
			"tags":           append([]string(nil), chunk.Tags...),
			"aliases":        append([]string(nil), chunk.Aliases...),
			"position":       chunk.Position,
			"chunk_strategy": chunk.ChunkStrategy,
		},
		Embedding: append([]float32(nil), embedding...),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
