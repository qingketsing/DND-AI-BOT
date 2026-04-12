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
}
