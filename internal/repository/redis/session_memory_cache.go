package redis

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
)

// SessionMemoryCache 定义会话长期记忆在 Redis 中的缓存接口。
type SessionMemoryCache interface {
	GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)
	Set(ctx context.Context, memory *model.SessionMemory, ttl time.Duration) error
	SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error
	DeleteBySessionID(ctx context.Context, sessionID string) error
}
