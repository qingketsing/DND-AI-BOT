package repository

import (
	"context"

	"DND-AI-BOT/internal/model"
)

// SessionRepository 定义会话聚合对上层暴露的统一存取接口。
type SessionRepository interface {
	Save(ctx context.Context, session *model.Session) error
	Load(ctx context.Context, sessionID string) (*model.Session, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.Session, error)
	Delete(ctx context.Context, sessionID string) error
}

// AsyncMessageEnqueueRepository 定义异步消息入队的事务化持久化契约。
type AsyncMessageEnqueueRepository interface {
	EnqueueAsyncMessage(ctx context.Context, session *model.Session, job model.MessageJob, event model.OutboxEvent) error
}
