package repository

import (
	"context"

	"DND-AI-BOT/internal/game/state"
)

// GameStateRepository 定义游戏进度聚合的统一存取接口。
type GameStateRepository interface {
	Save(ctx context.Context, state *state.GameState) error
	LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
}
