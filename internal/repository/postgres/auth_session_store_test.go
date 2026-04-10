package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestPGAuthSessionStoreSaveAndLoadByTokenHash(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGAuthSessionStore(db)

	now := time.Date(2026, 4, 10, 8, 30, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-1",
		Email:        "alice@example.com",
		PasswordHash: "hash-1",
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := NewPGUserStore(db).Save(context.Background(), user); err != nil {
		t.Fatalf("expected user save to succeed, got %v", err)
	}

	userAgent := "Mozilla/5.0"
	ipAddress := "127.0.0.1"
	session := &model.AuthSession{
		ID:        "session-1",
		UserID:    user.ID,
		TokenHash: "token-hash-1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
		UserAgent: &userAgent,
		IPAddress: &ipAddress,
	}

	if err := store.Save(context.Background(), session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	got, err := store.LoadByTokenHash(context.Background(), "token-hash-1")
	if err != nil {
		t.Fatalf("expected load by token hash to succeed, got %v", err)
	}
	if got.ID != session.ID || got.UserID != session.UserID || got.TokenHash != session.TokenHash {
		t.Fatalf("expected loaded session to match saved session, got %+v", got)
	}
	if got.UserAgent == nil || *got.UserAgent != userAgent {
		t.Fatalf("expected user agent %q, got %+v", userAgent, got.UserAgent)
	}
}

func TestPGAuthSessionStoreLoadByTokenHashReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGAuthSessionStore(db)

	_, err := store.LoadByTokenHash(context.Background(), "missing")
	if !errors.Is(err, repository.ErrAuthSessionNotFound) {
		t.Fatalf("expected ErrAuthSessionNotFound, got %v", err)
	}
}

func TestPGAuthSessionStoreRevoke(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGAuthSessionStore(db)

	now := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-3",
		Email:        "carol@example.com",
		PasswordHash: "hash-3",
		DisplayName:  "Carol",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := NewPGUserStore(db).Save(context.Background(), user); err != nil {
		t.Fatalf("expected user save to succeed, got %v", err)
	}

	session := &model.AuthSession{
		ID:        "session-2",
		UserID:    user.ID,
		TokenHash: "token-hash-2",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	revokeAt := now.Add(2 * time.Hour)
	if err := store.Revoke(context.Background(), session.ID, revokeAt); err != nil {
		t.Fatalf("expected revoke to succeed, got %v", err)
	}

	got, err := store.LoadByTokenHash(context.Background(), session.TokenHash)
	if err != nil {
		t.Fatalf("expected load after revoke to succeed, got %v", err)
	}
	if got.RevokedAt == nil || !got.RevokedAt.Equal(revokeAt) {
		t.Fatalf("expected revoked_at %v, got %+v", revokeAt, got.RevokedAt)
	}
}

func TestPGAuthSessionStoreRevokeReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGAuthSessionStore(db)

	err := store.Revoke(context.Background(), "missing-session", time.Now().UTC())
	if !errors.Is(err, repository.ErrAuthSessionNotFound) {
		t.Fatalf("expected ErrAuthSessionNotFound, got %v", err)
	}
}
