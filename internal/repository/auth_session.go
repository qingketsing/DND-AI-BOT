package repository

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
)

// AuthSessionRepository defines the persistence contract for login sessions.
type AuthSessionRepository interface {
	Save(ctx context.Context, session *model.AuthSession) error
	LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error)
	Revoke(ctx context.Context, sessionID string, now time.Time) error
}
