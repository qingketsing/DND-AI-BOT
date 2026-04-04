package redis

import (
	"context"
	"time"

	"DND-AI-BOT/internal/game/combat"
)

// EncounterCache 定义战斗状态在 Redis 中的缓存接口。
type EncounterCache interface {
	GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error)
	Set(ctx context.Context, encounter *combat.Encounter, ttl time.Duration) error
	SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
}
