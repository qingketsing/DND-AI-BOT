package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

// PGAuthSessionStore persists auth sessions in PostgreSQL.
type PGAuthSessionStore struct {
	db *sql.DB
}

// NewPGAuthSessionStore creates a PostgreSQL-backed auth session store.
func NewPGAuthSessionStore(db *sql.DB) *PGAuthSessionStore {
	return &PGAuthSessionStore{db: db}
}

// Save inserts or updates an auth session record.
func (s *PGAuthSessionStore) Save(ctx context.Context, session *model.AuthSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (
			id, user_id, token_hash, expires_at, created_at, updated_at,
			last_seen_at, user_agent, ip_address, revoked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    token_hash = EXCLUDED.token_hash,
		    expires_at = EXCLUDED.expires_at,
		    updated_at = EXCLUDED.updated_at,
		    last_seen_at = EXCLUDED.last_seen_at,
		    user_agent = EXCLUDED.user_agent,
		    ip_address = EXCLUDED.ip_address,
		    revoked_at = EXCLUDED.revoked_at
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.CreatedAt, session.UpdatedAt, session.LastSeenAt, session.UserAgent, session.IPAddress, session.RevokedAt)
	return err
}

// LoadByTokenHash returns a session by hashed token.
func (s *PGAuthSessionStore) LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error) {
	var (
		session   model.AuthSession
		lastSeen  sql.NullTime
		userAgent sql.NullString
		ipAddress sql.NullString
		revokedAt sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, updated_at, last_seen_at, user_agent, ip_address, revoked_at
		FROM auth_sessions
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&lastSeen,
		&userAgent,
		&ipAddress,
		&revokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrAuthSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	if lastSeen.Valid {
		value := lastSeen.Time
		session.LastSeenAt = &value
	}
	if userAgent.Valid {
		value := userAgent.String
		session.UserAgent = &value
	}
	if ipAddress.Valid {
		value := ipAddress.String
		session.IPAddress = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		session.RevokedAt = &value
	}

	return &session, nil
}

// Revoke marks a session as revoked at the supplied timestamp.
func (s *PGAuthSessionStore) Revoke(ctx context.Context, sessionID string, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = $2,
		    updated_at = $2
		WHERE id = $1
	`, sessionID, now)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return repository.ErrAuthSessionNotFound
	}
	return nil
}
