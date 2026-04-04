package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/repository/memory"
)

func TestCreateSessionSavesSessionToRepository(t *testing.T) {
	sessionRepository := memory.NewSessionRepository()
	service := NewSessionService(sessionRepository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(ctx, model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected session id to be generated")
	}
	if session.Channel != model.ChannelWeb {
		t.Fatalf("expected session channel %q, got %q", model.ChannelWeb, session.Channel)
	}
	if !sessionRepository.Exists(session.ID) {
		t.Fatalf("expected repository to contain session %q", session.ID)
	}
}

func TestGetSessionReturnsNotFoundForMissingSession(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()

	_, err := service.GetSession(ctx, "missing")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSendMessageAppendsUserAndMockReply(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	updated, err := service.SendMessage(ctx, SendMessageInput{
		SessionID: session.ID,
		UserID:    "user-1",
		UserName:  "Alice",
		Content:   "hello",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected send message to succeed, got %v", err)
	}
	if len(updated.History) != 2 {
		t.Fatalf("expected 2 history records, got %d", len(updated.History))
	}
	if updated.History[0].Source != model.MessageSourceUser {
		t.Fatalf("expected first record source %q, got %q", model.MessageSourceUser, updated.History[0].Source)
	}
	if updated.History[1].Source != model.MessageSourceAgent {
		t.Fatalf("expected second record source %q, got %q", model.MessageSourceAgent, updated.History[1].Source)
	}
	if updated.History[1].Message.Content == "" {
		t.Fatal("expected mock reply content to be present")
	}
}

func TestSendMessageRejectsEmptyContent(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	_, err = service.SendMessage(ctx, SendMessageInput{
		SessionID: session.ID,
		UserID:    "user-1",
		UserName:  "Alice",
		Content:   "   ",
	}, now.Add(time.Minute))
	if !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got %v", err)
	}
}

func TestCreateSessionRejectsInvalidChannel(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	_, err := service.CreateSession(ctx, model.Channel("desktop"), now)
	if !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("expected ErrInvalidChannel, got %v", err)
	}
}
