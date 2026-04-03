package memory

import (
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
)

func TestSessionRepositorySaveAndLoad(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", model.ChannelWeb, now)
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	if err := repository.Save(session); err != nil {
		t.Fatalf("expected save to succeed, got error %v", err)
	}

	loaded, err := repository.Load("session-1")
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

	_, err := repository.Load("missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionRepositoryLoadReturnsIndependentSession(t *testing.T) {
	repository := NewSessionRepository()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", model.ChannelWeb, now)
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	if err := repository.Save(session); err != nil {
		t.Fatalf("expected save to succeed, got error %v", err)
	}

	loaded, err := repository.Load("session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got error %v", err)
	}
	loaded.AppendSystemMessage("mutated", now.Add(2*time.Minute))

	reloaded, err := repository.Load("session-1")
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
	session := model.NewSession("session-1", model.ChannelWeb, now)

	if err := repository.Save(session); err != nil {
		t.Fatalf("expected initial save to succeed, got error %v", err)
	}

	session.AppendAgentMessage(model.User{ID: "agent-1", Name: "DM Agent"}, "reply", now.Add(time.Minute))
	if err := repository.Save(session); err != nil {
		t.Fatalf("expected second save to succeed, got error %v", err)
	}

	loaded, err := repository.Load("session-1")
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
