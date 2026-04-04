package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
)

// PGGameStateStore 负责将游戏进度写入 PostgreSQL 并从中恢复。
type PGGameStateStore struct {
	db *sql.DB
}

// NewPGGameStateStore 创建基于 database/sql 的游戏进度 PG 存储实现。
func NewPGGameStateStore(db *sql.DB) *PGGameStateStore {
	return &PGGameStateStore{db: db}
}

// UpsertGameState 将游戏进度写入数据库，玩家状态整体序列化到 JSONB 字段。
func (s *PGGameStateStore) UpsertGameState(ctx context.Context, gameState *state.GameState) error {
	playerData, err := json.Marshal(gameState.Player)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO game_states (id, session_id, current_scene, player_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET session_id = EXCLUDED.session_id,
		    current_scene = EXCLUDED.current_scene,
		    player_data = EXCLUDED.player_data,
		    updated_at = EXCLUDED.updated_at
	`, gameState.ID, gameState.SessionID, gameState.CurrentScene, playerData, gameState.CreatedAt, gameState.UpdatedAt)
	return err
}

// GetGameStateBySessionID 按会话 ID 读取游戏进度并反序列化玩家状态。
func (s *PGGameStateStore) GetGameStateBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	var (
		id           string
		currentScene string
		playerData   []byte
		createdAt    sql.NullTime
		updatedAt    sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, current_scene, player_data, created_at, updated_at
		FROM game_states
		WHERE session_id = $1
	`, sessionID).Scan(&id, &currentScene, &playerData, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrGameStateNotFound
	}
	if err != nil {
		return nil, err
	}

	var player state.PlayerState
	if err := json.Unmarshal(playerData, &player); err != nil {
		return nil, err
	}

	return &state.GameState{
		ID:           id,
		SessionID:    sessionID,
		Player:       player,
		CurrentScene: currentScene,
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}, nil
}
