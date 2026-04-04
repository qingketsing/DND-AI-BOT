package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
)

// PGEncounterStore 负责将战斗状态写入 PostgreSQL 并从中恢复。
type PGEncounterStore struct {
	db *sql.DB
}

// NewPGEncounterStore 创建基于 database/sql 的战斗 PG 存储实现。
func NewPGEncounterStore(db *sql.DB) *PGEncounterStore {
	return &PGEncounterStore{db: db}
}

// UpsertEncounter 将战斗主字段和完整快照一起写入数据库。
func (s *PGEncounterStore) UpsertEncounter(ctx context.Context, encounter *combat.Encounter) error {
	encounterData, err := json.Marshal(encounter)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO encounters (id, session_id, round, turn_index, encounter_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		SET session_id = EXCLUDED.session_id,
		    round = EXCLUDED.round,
		    turn_index = EXCLUDED.turn_index,
		    encounter_data = EXCLUDED.encounter_data,
		    updated_at = EXCLUDED.updated_at
	`, encounter.ID, encounter.SessionID, encounter.Round, encounter.TurnIndex, encounterData, encounter.StartedAt, encounter.UpdatedAt)
	return err
}

// GetEncounterBySessionID 按会话 ID 读取战斗快照并恢复领域对象。
func (s *PGEncounterStore) GetEncounterBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	var encounterData []byte

	err := s.db.QueryRowContext(ctx, `
		SELECT encounter_data
		FROM encounters
		WHERE session_id = $1
	`, sessionID).Scan(&encounterData)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrEncounterNotFound
	}
	if err != nil {
		return nil, err
	}

	var encounter combat.Encounter
	if err := json.Unmarshal(encounterData, &encounter); err != nil {
		return nil, err
	}

	return &encounter, nil
}
