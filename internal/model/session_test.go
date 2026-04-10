package model

import (
	"testing"
	"time"
)

func TestNewSessionInitializesTimestampsAndEmptyHistory(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	session := NewSession("session-1", ChannelWeb, now)

	if session.ID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", session.ID)
	}
	if session.Channel != ChannelWeb {
		t.Fatalf("expected session channel %q, got %q", ChannelWeb, session.Channel)
	}
	if len(session.History) != 0 {
		t.Fatalf("expected empty history, got %d records", len(session.History))
	}
	if !session.CreatedAt.Equal(now) {
		t.Fatalf("expected CreatedAt %v, got %v", now, session.CreatedAt)
	}
	if !session.UpdatedAt.Equal(now) {
		t.Fatalf("expected UpdatedAt %v, got %v", now, session.UpdatedAt)
	}
}

func TestAppendUserMessageAddsRecordAndUpdatesSession(t *testing.T) {
	createdAt := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	messageAt := createdAt.Add(2 * time.Minute)
	session := NewSession("session-1", ChannelWeb, createdAt)
	user := SessionUser{ID: "user-1", Name: "Alice"}

	record := session.AppendUserMessage(user, "hello", messageAt)

	if len(session.History) != 1 {
		t.Fatalf("expected 1 record, got %d", len(session.History))
	}
	if record.ID != "user-1" {
		t.Fatalf("expected record id user-1, got %q", record.ID)
	}
	if record.Sequence != 1 {
		t.Fatalf("expected sequence 1, got %d", record.Sequence)
	}
	if record.Source != MessageSourceUser {
		t.Fatalf("expected source %q, got %q", MessageSourceUser, record.Source)
	}
	if record.User != user {
		t.Fatalf("expected user %+v, got %+v", user, record.User)
	}
	if record.Message.Content != "hello" {
		t.Fatalf("expected content hello, got %q", record.Message.Content)
	}
	if !record.CreatedAt.Equal(messageAt) {
		t.Fatalf("expected record CreatedAt %v, got %v", messageAt, record.CreatedAt)
	}
	if !session.UpdatedAt.Equal(messageAt) {
		t.Fatalf("expected UpdatedAt %v, got %v", messageAt, session.UpdatedAt)
	}
}

func TestAppendAgentAndSystemMessagesUseExpectedSources(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := NewSession("session-1", ChannelWeb, now)
	agent := SessionUser{ID: "agent-1", Name: "DM Agent"}

	agentRecord := session.AppendAgentMessage(agent, "agent reply", now.Add(time.Minute))
	systemRecord := session.AppendSystemMessage("system note", now.Add(2*time.Minute))

	if agentRecord.Source != MessageSourceAgent {
		t.Fatalf("expected agent source %q, got %q", MessageSourceAgent, agentRecord.Source)
	}
	if systemRecord.Source != MessageSourceSystem {
		t.Fatalf("expected system source %q, got %q", MessageSourceSystem, systemRecord.Source)
	}
	if systemRecord.User.ID != "system" || systemRecord.User.Name != "system" {
		t.Fatalf("expected system user, got %+v", systemRecord.User)
	}
	if systemRecord.Sequence != 2 {
		t.Fatalf("expected sequence 2, got %d", systemRecord.Sequence)
	}
}

func TestLastRecordReturnsLatestHistoryRecord(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := NewSession("session-1", ChannelWeb, now)

	if _, ok := session.LastRecord(); ok {
		t.Fatal("expected no last record for empty session")
	}

	session.AppendUserMessage(SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	last, ok := session.LastRecord()
	if !ok {
		t.Fatal("expected last record after append")
	}
	if last.Sequence != 1 {
		t.Fatalf("expected sequence 1, got %d", last.Sequence)
	}
}

func TestHistoryRecordsReturnsCopy(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := NewSession("session-1", ChannelWeb, now)
	session.AppendUserMessage(SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	history := session.HistoryRecords()
	history[0].Message.Content = "mutated"

	if session.History[0].Message.Content != "hello" {
		t.Fatalf("expected session history to remain unchanged, got %q", session.History[0].Message.Content)
	}
}
