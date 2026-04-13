package repository

import (
	"context"
	"errors"

	"DND-AI-BOT/internal/model"
)

// ErrSessionMemoryNotFound 表示指定会话还没有长期记忆记录。
var ErrSessionMemoryNotFound = errors.New("session memory not found")

// SessionMemoryRepository 定义会话长期记忆的读写接口。
type SessionMemoryRepository interface {
	LoadBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)
	Save(ctx context.Context, memory *model.SessionMemory) error
}
