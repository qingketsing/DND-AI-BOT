package postgres

import (
	"context"
	"database/sql"
	"errors"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

// PGUserStore persists authenticated users in PostgreSQL.
type PGUserStore struct {
	db *sql.DB
}

// NewPGUserStore creates a PostgreSQL-backed user store.
func NewPGUserStore(db *sql.DB) *PGUserStore {
	return &PGUserStore{db: db}
}

// Save inserts or updates a user record.
func (s *PGUserStore) Save(ctx context.Context, user *model.User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (
			id, email, password_hash, display_name, status,
			created_at, updated_at, last_login_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE
		SET email = EXCLUDED.email,
		    password_hash = EXCLUDED.password_hash,
		    display_name = EXCLUDED.display_name,
		    status = EXCLUDED.status,
		    updated_at = EXCLUDED.updated_at,
		    last_login_at = EXCLUDED.last_login_at
	`, user.ID, user.Email, user.PasswordHash, user.DisplayName, string(user.Status), user.CreatedAt, user.UpdatedAt, user.LastLoginAt)
	return err
}

// LoadByID returns a user by primary key.
func (s *PGUserStore) LoadByID(ctx context.Context, id string) (*model.User, error) {
	return s.loadOne(ctx, `
		SELECT id, email, password_hash, display_name, status, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`, id)
}

// LoadByEmail returns a user by email address.
func (s *PGUserStore) LoadByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.loadOne(ctx, `
		SELECT id, email, password_hash, display_name, status, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`, email)
}

func (s *PGUserStore) loadOne(ctx context.Context, query string, arg any) (*model.User, error) {
	var (
		user      model.User
		status    string
		lastLogin sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, query, arg).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLogin,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	user.Status = model.UserStatus(status)
	if lastLogin.Valid {
		value := lastLogin.Time
		user.LastLoginAt = &value
	}

	return &user, nil
}
