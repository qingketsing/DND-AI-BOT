package redis

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
)

// SessionCache 定义会话在 Redis 中的缓存读写接口。
type SessionCache interface {
	Get(ctx context.Context, sessionID string) (*model.Session, error)
	Set(ctx context.Context, session *model.Session, ttl time.Duration) error
	SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error
	Delete(ctx context.Context, sessionID string) error
}
