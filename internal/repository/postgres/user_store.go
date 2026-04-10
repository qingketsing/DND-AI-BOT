package postgres

import (
	"context"

	"DND-AI-BOT/internal/model"
)

// UserStore defines PostgreSQL persistence for authenticated users.
type UserStore interface {
	Save(ctx context.Context, user *model.User) error
	LoadByID(ctx context.Context, id string) (*model.User, error)
	LoadByEmail(ctx context.Context, email string) (*model.User, error)
}
