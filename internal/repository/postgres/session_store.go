package postgres

import (
	"context"

	"DND-AI-BOT/internal/model"
)

// SessionStore 定义 PostgreSQL 真相源需要实现的会话存取能力。
type SessionStore interface {
	UpsertSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	ListSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}
