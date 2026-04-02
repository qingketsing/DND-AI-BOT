package service

import (
	"errors"
	"testing"
	"time"

	"../model"
	"../repository/memory"
)

func TestCreateSessionSavesSessionToRepository(t *testing.T) {
	repository := memory.NewSessionRepository()
	service := NewSessionService(repository)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected session id to be generated")
	}
	if session.Channel != model.ChannelWeb {
		t.Fatalf("expected session channel %q, got %q", model.ChannelWeb, session.Channel)
	}
	if !repository.Exists(session.ID) {
		t.Fatalf("expected repository to contain session %q", session.ID)
	}
}

func TestGetSessionReturnsNotFoundForMissingSession(t *testing.T) {
	repository := memory.NewSessionRepository()
	service := NewSessionService(repository)

	_, err := service.GetSession("missing")
	if !errors.Is(err, memory.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSendMessageAppendsUserAndMockReply(t *testing.T) {
	repository := memory.NewSessionRepository()
	service := NewSessionService(repository)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	updated, err := service.SendMessage(SendMessageInput{
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
	repository := memory.NewSessionRepository()
	service := NewSessionService(repository)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	_, err = service.SendMessage(SendMessageInput{
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
	repository := memory.NewSessionRepository()
	service := NewSessionService(repository)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	_, err := service.CreateSession(model.Channel("desktop"), now)
	if !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("expected ErrInvalidChannel, got %v", err)
	}
}
