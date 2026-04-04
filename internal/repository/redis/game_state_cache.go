package redis

import (
	"context"
	"time"

	"DND-AI-BOT/internal/game/state"
)

// GameStateCache 定义游戏进度在 Redis 中的缓存接口。
type GameStateCache interface {
	GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
	Set(ctx context.Context, state *state.GameState, ttl time.Duration) error
	SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
}
