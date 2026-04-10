package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestPGUserStoreSaveAndLoadByID(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGUserStore(db)

	now := time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-1",
		Email:        "alice@example.com",
		PasswordHash: "hash-1",
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.Save(context.Background(), user); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	got, err := store.LoadByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected load by id to succeed, got %v", err)
	}
	if got.Email != user.Email || got.PasswordHash != user.PasswordHash || got.DisplayName != user.DisplayName {
		t.Fatalf("expected loaded user to match saved user, got %+v", got)
	}
}

func TestPGUserStoreLoadByEmailReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGUserStore(db)

	_, err := store.LoadByEmail(context.Background(), "missing@example.com")
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestPGUserStoreLoadByEmail(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGUserStore(db)

	now := time.Date(2026, 4, 10, 8, 15, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-2",
		Email:        "bob@example.com",
		PasswordHash: "hash-2",
		DisplayName:  "Bob",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Save(context.Background(), user); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	got, err := store.LoadByEmail(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("expected load by email to succeed, got %v", err)
	}
	if got.ID != user.ID || got.Email != user.Email {
		t.Fatalf("expected loaded user to match saved user, got %+v", got)
	}
}
