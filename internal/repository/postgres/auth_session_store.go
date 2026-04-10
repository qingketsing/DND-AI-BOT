package postgres

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
)

// AuthSessionStore defines PostgreSQL persistence for login sessions.
type AuthSessionStore interface {
	Save(ctx context.Context, session *model.AuthSession) error
	LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error)
	Revoke(ctx context.Context, sessionID string, now time.Time) error
}
