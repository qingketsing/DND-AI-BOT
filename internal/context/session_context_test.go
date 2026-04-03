package context

import (
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository/memory"
)

func TestGetSessionReturnsStoredSession(t *testing.T) {
	store, session := newTestSessionContextStore(t)

	got, err := store.GetSession(session.ID)
	if err != nil {
		t.Fatalf("expected get session to succeed, got %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, got.ID)
	}
}

func TestGetRecentRecordsReturnsLastNRecords(t *testing.T) {
	store, session := newTestSessionContextStore(t)

	records, err := store.GetRecentRecords(session.ID, 2)
	if err != nil {
		t.Fatalf("expected get recent records to succeed, got %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Sequence != 2 || records[1].Sequence != 3 {
		t.Fatalf("expected sequences 2 and 3, got %d and %d", records[0].Sequence, records[1].Sequence)
	}
}

func TestGetLastRecordReturnsLatestRecord(t *testing.T) {
	store, session := newTestSessionContextStore(t)

	record, ok, err := store.GetLastRecord(session.ID)
	if err != nil {
		t.Fatalf("expected get last record to succeed, got %v", err)
	}
	if !ok {
		t.Fatal("expected last record to exist")
	}
	if record.Sequence != 3 {
		t.Fatalf("expected sequence 3, got %d", record.Sequence)
	}
}

func TestGetChannelReturnsSessionChannel(t *testing.T) {
	store, session := newTestSessionContextStore(t)

	channel, err := store.GetChannel(session.ID)
	if err != nil {
		t.Fatalf("expected get channel to succeed, got %v", err)
	}
	if channel != session.Channel {
		t.Fatalf("expected channel %q, got %q", session.Channel, channel)
	}
}

func newTestSessionContextStore(t *testing.T) (*DefaultSessionContextStore, *model.Session) {
	t.Helper()

	repository := memory.NewSessionRepository()
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", model.ChannelWeb, now)
	session.AppendSystemMessage("system", now.Add(time.Minute))
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Alice"}, "hello", now.Add(2*time.Minute))
	session.AppendAgentMessage(model.User{ID: "agent-1", Name: "DM Agent"}, "reply", now.Add(3*time.Minute))

	if err := repository.Save(session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	return NewSessionContextStore(repository), session
}
