package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestPGSessionStoreUpsertAndGetSession(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGSessionStore(db)

	now := time.Date(2026, 4, 12, 8, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	session.Title = "测试会话"
	session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	if err := store.UpsertSession(context.Background(), session); err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	got, err := store.GetSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("expected get session to succeed, got %v", err)
	}
	if got.UserID != session.UserID || got.Title != session.Title {
		t.Fatalf("expected session ownership fields to round-trip, got %+v", got)
	}
	if len(got.History) != 1 || got.History[0].Message.Content != "hello" {
		t.Fatalf("expected history to round-trip, got %+v", got.History)
	}
}

func TestPGSessionStoreGetSessionReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGSessionStore(db)

	_, err := store.GetSession(context.Background(), "missing")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestPGSessionStoreListSessionsByUserID(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGSessionStore(db)

	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	session1 := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	session2 := model.NewSession("session-2", "user-2", model.ChannelWeb, now.Add(time.Minute))
	session3 := model.NewSession("session-3", "user-1", model.ChannelBot, now.Add(2*time.Minute))

	for _, session := range []*model.Session{session1, session2, session3} {
		if err := store.UpsertSession(context.Background(), session); err != nil {
			t.Fatalf("expected upsert to succeed, got %v", err)
		}
	}

	got, err := store.ListSessionsByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected list by user to succeed, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions for user-1, got %d", len(got))
	}
	for _, session := range got {
		if session.UserID != "user-1" {
			t.Fatalf("expected listed session to belong to user-1, got %q", session.UserID)
		}
	}
}

func TestPGSessionStoreDeleteSession(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGSessionStore(db)

	now := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	if err := store.UpsertSession(context.Background(), session); err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}
	if err := store.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("expected delete session to succeed, got %v", err)
	}

	_, err := store.GetSession(context.Background(), session.ID)
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestPGSessionStoreDeleteSessionReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGSessionStore(db)

	err := store.DeleteSession(context.Background(), "missing")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
