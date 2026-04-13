package service

import (
	"context"
	"errors"
	"testing"
	"time"

	agentprompt "DND-AI-BOT/internal/agent/prompt"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/repository/memory"
)

func TestCreateSessionSavesSessionToRepository(t *testing.T) {
	sessionRepository := memory.NewSessionRepository()
	service := NewSessionService(sessionRepository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected session id to be generated")
	}
	if session.Channel != model.ChannelWeb {
		t.Fatalf("expected session channel %q, got %q", model.ChannelWeb, session.Channel)
	}
	if session.UserID != "user-1" {
		t.Fatalf("expected session user id %q, got %q", "user-1", session.UserID)
	}
	if !sessionRepository.Exists(session.ID) {
		t.Fatalf("expected repository to contain session %q", session.ID)
	}
}

func TestGetSessionForUserReturnsNotFoundForMissingSession(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()

	_, err := service.GetSessionForUser(ctx, "user-1", "missing")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSendMessageAppendsUserAndMockReply(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	updated, err := service.SendMessage(ctx, "user-1", "Alice", SendMessageInput{
		SessionID: session.ID,
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

func TestSendMessageUsesAgentServiceReplyWhenConfigured(t *testing.T) {
	repository := memory.NewSessionRepository()
	agentService := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{
			Reply: "你当前背包里有一瓶治疗药水。",
		}, nil
	}, nil)
	service := NewSessionService(repository, agentService)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	updated, err := service.SendMessage(ctx, "user-1", "Alice", SendMessageInput{
		SessionID: session.ID,
		Content:   "我的背包里有什么？",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected send message to succeed, got %v", err)
	}

	if got := updated.History[1].Message.Content; got != "你当前背包里有一瓶治疗药水。" {
		t.Fatalf("expected agent runtime reply, got %q", got)
	}
}

func TestSendMessageRefreshesSessionMemoryAfterReply(t *testing.T) {
	repository := memory.NewSessionRepository()
	agentService := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{Reply: "规则裁定完成。"}, nil
	}, nil)
	refresher := &fakeSessionMemoryRefresher{}
	service := NewSessionService(repository, agentService)
	service.SetMemoryRefresher(refresher)
	ctx := context.Background()
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	initialCalls := refresher.calls

	_, err = service.SendMessage(ctx, "user-1", "Alice", SendMessageInput{
		SessionID: session.ID,
		Content:   "hello",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected send message to succeed, got %v", err)
	}
	if refresher.calls != initialCalls+1 || refresher.lastSessionID != session.ID {
		t.Fatalf("expected refresher to be called once for session %q, got %+v", session.ID, refresher)
	}
}

func TestSendMessagePassesDefaultSystemPromptToAgent(t *testing.T) {
	repository := memory.NewSessionRepository()
	var captured AgentReplyInput
	agentService := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		captured = input
		return AgentReplyResult{Reply: "规则裁定完成。"}, nil
	}, nil)
	service := NewSessionService(repository, agentService)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	_, err = service.SendMessage(ctx, "user-1", "Alice", SendMessageInput{
		SessionID: session.ID,
		Content:   "法师怎么准备法术？",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected send message to succeed, got %v", err)
	}

	if captured.SystemPrompt != agentprompt.DefaultSystemPrompt {
		t.Fatalf("expected default system prompt to be passed to agent, got %q", captured.SystemPrompt)
	}
}

func TestSendMessageRejectsEmptyContent(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	_, err = service.SendMessage(ctx, "user-1", "Alice", SendMessageInput{
		SessionID: session.ID,
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

	_, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.Channel("desktop"),
	}, now)
	if !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("expected ErrInvalidChannel, got %v", err)
	}
}

func TestGetSessionForUserReturnsForbiddenForDifferentOwner(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(ctx, CreateSessionInput{
		UserID:  "user-1",
		Channel: model.ChannelWeb,
	}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	_, err = service.GetSessionForUser(ctx, "user-2", session.ID)
	if !errors.Is(err, ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden, got %v", err)
	}
}

func TestListSessionsReturnsOnlyCurrentUserSessions(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	_, _ = service.CreateSession(ctx, CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	_, _ = service.CreateSession(ctx, CreateSessionInput{UserID: "user-2", Channel: model.ChannelWeb}, now.Add(time.Minute))
	_, _ = service.CreateSession(ctx, CreateSessionInput{UserID: "user-1", Channel: model.ChannelBot}, now.Add(2*time.Minute))

	sessions, err := service.ListSessions(ctx, "user-1")
	if err != nil {
		t.Fatalf("expected list sessions to succeed, got %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions for user-1, got %d", len(sessions))
	}
	for _, session := range sessions {
		if session.UserID != "user-1" {
			t.Fatalf("expected session user id %q, got %q", "user-1", session.UserID)
		}
	}
}

func TestSendMessageRejectsDifferentOwner(t *testing.T) {
	service := NewSessionService(memory.NewSessionRepository())
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(ctx, CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	_, err = service.SendMessage(ctx, "user-2", "Bob", SendMessageInput{
		SessionID: session.ID,
		Content:   "hello",
	}, now.Add(time.Minute))
	if !errors.Is(err, ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden, got %v", err)
	}
}

func TestDeleteSessionDeletesOwnedSession(t *testing.T) {
	sessionRepository := memory.NewSessionRepository()
	service := NewSessionService(sessionRepository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(ctx, CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	if err := service.DeleteSession(ctx, "user-1", session.ID); err != nil {
		t.Fatalf("expected delete session to succeed, got %v", err)
	}

	_, err = service.GetSessionForUser(ctx, "user-1", session.ID)
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestDeleteSessionRejectsDifferentOwner(t *testing.T) {
	repository := memory.NewSessionRepository()
	service := NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session, err := service.CreateSession(ctx, CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}

	err = service.DeleteSession(ctx, "user-2", session.ID)
	if !errors.Is(err, ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden, got %v", err)
	}
}

type fakeSessionMemoryRefresher struct {
	calls         int
	lastSessionID string
	lastNow       time.Time
	err           error
}

func (f *fakeSessionMemoryRefresher) RefreshIfNeeded(ctx context.Context, sessionID string, now time.Time) error {
	_ = ctx
	f.calls++
	f.lastSessionID = sessionID
	f.lastNow = now
	return f.err
}
