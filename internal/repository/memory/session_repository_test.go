package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
)

func TestSessionRepositorySaveAndLoad(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	if err := repository.Save(context.Background(), session); err != nil {
		t.Fatalf("expected save to succeed, got error %v", err)
	}

	loaded, err := repository.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got error %v", err)
	}
	if loaded.ID != session.ID {
		t.Fatalf("expected loaded id %q, got %q", session.ID, loaded.ID)
	}
	if len(loaded.History) != 1 {
		t.Fatalf("expected loaded history length 1, got %d", len(loaded.History))
	}
	if loaded.History[0].Message.Content != "hello" {
		t.Fatalf("expected loaded content hello, got %q", loaded.History[0].Message.Content)
	}
}

func TestSessionRepositoryLoadMissingSession(t *testing.T) {
	repository := NewSessionRepository()

	_, err := repository.Load(context.Background(), "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionRepositoryLoadReturnsIndependentSession(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	if err := repository.Save(context.Background(), session); err != nil {
		t.Fatalf("expected save to succeed, got error %v", err)
	}

	loaded, err := repository.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got error %v", err)
	}
	loaded.AppendSystemMessage("mutated", now.Add(2*time.Minute))

	reloaded, err := repository.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected second load to succeed, got error %v", err)
	}
	if len(reloaded.History) != 1 {
		t.Fatalf("expected reloaded history length 1, got %d", len(reloaded.History))
	}
}

func TestSessionRepositorySaveOverridesExistingSnapshot(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)

	if err := repository.Save(context.Background(), session); err != nil {
		t.Fatalf("expected initial save to succeed, got error %v", err)
	}

	session.AppendAgentMessage(model.SessionUser{ID: "agent-1", Name: "DM Agent"}, "reply", now.Add(time.Minute))
	if err := repository.Save(context.Background(), session); err != nil {
		t.Fatalf("expected second save to succeed, got error %v", err)
	}

	loaded, err := repository.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got error %v", err)
	}
	if len(loaded.History) != 1 {
		t.Fatalf("expected loaded history length 1, got %d", len(loaded.History))
	}
	if loaded.History[0].Source != model.MessageSourceAgent {
		t.Fatalf("expected loaded source %q, got %q", model.MessageSourceAgent, loaded.History[0].Source)
	}
}

func TestSessionRepositoryListByUserID(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session1 := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	session2 := model.NewSession("session-2", "user-2", model.ChannelWeb, now.Add(time.Minute))
	session3 := model.NewSession("session-3", "user-1", model.ChannelBot, now.Add(2*time.Minute))

	for _, session := range []*model.Session{session1, session2, session3} {
		if err := repository.Save(context.Background(), session); err != nil {
			t.Fatalf("expected save to succeed, got %v", err)
		}
	}

	loaded, err := repository.ListByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected list by user to succeed, got %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 sessions for user-1, got %d", len(loaded))
	}
	for _, session := range loaded {
		if session.UserID != "user-1" {
			t.Fatalf("expected listed session to belong to user-1, got %q", session.UserID)
		}
	}
}

func TestSessionRepositoryDelete(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)

	if err := repository.Save(context.Background(), session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	if err := repository.Delete(context.Background(), session.ID); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}

	_, err := repository.Load(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestSessionRepositoryDeleteMissingSession(t *testing.T) {
	repository := NewSessionRepository()

	err := repository.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
