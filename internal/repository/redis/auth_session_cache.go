package redis

import (
	"context"
	"time"
)

// CachedAuthSession is the Redis projection of an auth session.
type CachedAuthSession struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

// AuthSessionCache defines the Redis cache contract for auth session lookups.
type AuthSessionCache interface {
	Set(ctx context.Context, tokenHash string, session CachedAuthSession, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (CachedAuthSession, error)
	Delete(ctx context.Context, tokenHash string) error
}
