package repository

import (
	"context"

	"DND-AI-BOT/internal/model"
)

// UserRepository defines the persistence contract for authenticated users.
type UserRepository interface {
	Save(ctx context.Context, user *model.User) error
	LoadByID(ctx context.Context, id string) (*model.User, error)
	LoadByEmail(ctx context.Context, email string) (*model.User, error)
}
