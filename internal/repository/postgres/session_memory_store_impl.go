package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

// PGSessionMemoryStore 负责将会话长期记忆写入 PostgreSQL 并从中恢复。
type PGSessionMemoryStore struct {
	db *sql.DB
}

// NewPGSessionMemoryStore 创建基于 database/sql 的会话长期记忆 PG 存储实现。
func NewPGSessionMemoryStore(db *sql.DB) *PGSessionMemoryStore {
	return &PGSessionMemoryStore{db: db}
}

// SaveSessionMemory 将会话长期记忆写入数据库。
func (s *PGSessionMemoryStore) SaveSessionMemory(ctx context.Context, memory *model.SessionMemory) error {
	recentKeyEvents, err := json.Marshal(memory.RecentKeyEvents)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO session_memories (
			session_id,
			character_summary,
			scene_summary,
			current_objective,
			recent_key_events,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (session_id) DO UPDATE
		SET character_summary = EXCLUDED.character_summary,
		    scene_summary = EXCLUDED.scene_summary,
		    current_objective = EXCLUDED.current_objective,
		    recent_key_events = EXCLUDED.recent_key_events,
		    updated_at = EXCLUDED.updated_at
	`, memory.SessionID, memory.CharacterSummary, memory.SceneSummary, memory.CurrentObjective, recentKeyEvents, memory.UpdatedAt)
	return err
}

// GetSessionMemoryBySessionID 按会话 ID 读取长期记忆。
func (s *PGSessionMemoryStore) GetSessionMemoryBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	var (
		characterSummary string
		sceneSummary     string
		currentObjective string
		recentKeyEvents  []byte
		updatedAt        sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT character_summary, scene_summary, current_objective, recent_key_events, updated_at
		FROM session_memories
		WHERE session_id = $1
	`, sessionID).Scan(&characterSummary, &sceneSummary, &currentObjective, &recentKeyEvents, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrSessionMemoryNotFound
	}
	if err != nil {
		return nil, err
	}

	var events []string
	if len(recentKeyEvents) > 0 {
		if err := json.Unmarshal(recentKeyEvents, &events); err != nil {
			return nil, err
		}
	}

	return &model.SessionMemory{
		SessionID:        sessionID,
		CharacterSummary: characterSummary,
		SceneSummary:     sceneSummary,
		CurrentObjective: currentObjective,
		RecentKeyEvents:  events,
		UpdatedAt:        updatedAt.Time,
	}, nil
}

// DeleteSessionMemoryBySessionID 按会话 ID 删除长期记忆；不存在时也视为清理成功。
func (s *PGSessionMemoryStore) DeleteSessionMemoryBySessionID(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM session_memories WHERE session_id = $1`, sessionID)
	return err
}
