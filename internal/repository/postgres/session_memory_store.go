package postgres

import (
	"context"

	"DND-AI-BOT/internal/model"
)

// SessionMemoryStore 定义 PostgreSQL 会话长期记忆真相源接口。
type SessionMemoryStore interface {
	SaveSessionMemory(ctx context.Context, memory *model.SessionMemory) error
	GetSessionMemoryBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)
	DeleteSessionMemoryBySessionID(ctx context.Context, sessionID string) error
}
