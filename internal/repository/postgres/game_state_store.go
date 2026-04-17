package postgres

import (
	"context"

	"DND-AI-BOT/internal/game/state"
)

// GameStateStore 定义 PostgreSQL 游戏进度真相源接口。
type GameStateStore interface {
	UpsertGameState(ctx context.Context, state *state.GameState) error
	GetGameStateBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
	DeleteGameStateBySessionID(ctx context.Context, sessionID string) error
}
