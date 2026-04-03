package context

import (
	"testing"
	"time"

	basecontext "DND-AI-BOT/internal/context"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository/memory"
)

func TestBuildContextReturnsSessionMetadataAndRecentRecords(t *testing.T) {
	provider, session := newTestProvider(t)

	result, err := provider.BuildContext(session.ID, 2)
	if err != nil {
		t.Fatalf("expected build context to succeed, got %v", err)
	}
	if result.SessionID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, result.SessionID)
	}
	if result.Channel != model.ChannelBot {
		t.Fatalf("expected channel %q, got %q", model.ChannelBot, result.Channel)
	}
	if len(result.RecentRecords) != 2 {
		t.Fatalf("expected 2 recent records, got %d", len(result.RecentRecords))
	}
}

func TestBuildContextReturnsNilLastRecordForEmptySession(t *testing.T) {
	repository := memory.NewSessionRepository()
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("empty-session", model.ChannelWeb, now)
	if err := repository.Save(session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	store := basecontext.NewSessionContextStore(repository)
	provider := NewProvider(store)

	result, err := provider.BuildContext(session.ID, 5)
	if err != nil {
		t.Fatalf("expected build context to succeed, got %v", err)
	}
	if result.LastRecord != nil {
		t.Fatal("expected last record to be nil for empty session")
	}
}

func TestBuildContextReturnsLastRecordWhenHistoryExists(t *testing.T) {
	provider, session := newTestProvider(t)

	result, err := provider.BuildContext(session.ID, 5)
	if err != nil {
		t.Fatalf("expected build context to succeed, got %v", err)
	}
	if result.LastRecord == nil {
		t.Fatal("expected last record to exist")
	}
	if result.LastRecord.Sequence != 3 {
		t.Fatalf("expected sequence 3, got %d", result.LastRecord.Sequence)
	}
}

func newTestProvider(t *testing.T) (*DefaultProvider, *model.Session) {
	t.Helper()

	repository := memory.NewSessionRepository()
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-bot", model.ChannelBot, now)
	session.AppendSystemMessage("system", now.Add(time.Minute))
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Alice"}, "hello", now.Add(2*time.Minute))
	session.AppendAgentMessage(model.User{ID: "agent-1", Name: "DM Agent"}, "reply", now.Add(3*time.Minute))

	if err := repository.Save(session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	store := basecontext.NewSessionContextStore(repository)
	return NewProvider(store), session
}
