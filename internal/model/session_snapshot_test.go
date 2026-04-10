package model

import (
	"testing"
	"time"
)

func TestSessionToSnapshotCopiesSessionData(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := NewSession("session-1", ChannelBot, now)
	session.AppendUserMessage(SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	session.AppendSystemMessage("system note", now.Add(2*time.Minute))

	snapshot := session.ToSnapshot()

	if snapshot.ID != session.ID {
		t.Fatalf("expected snapshot id %q, got %q", session.ID, snapshot.ID)
	}
	if snapshot.Channel != ChannelBot {
		t.Fatalf("expected snapshot channel %q, got %q", ChannelBot, snapshot.Channel)
	}
	if len(snapshot.History) != 2 {
		t.Fatalf("expected 2 snapshot records, got %d", len(snapshot.History))
	}
	if snapshot.History[0].Message.Content != "hello" {
		t.Fatalf("expected first snapshot content hello, got %q", snapshot.History[0].Message.Content)
	}
	if snapshot.History[1].Source != MessageSourceSystem {
		t.Fatalf("expected second snapshot source %q, got %q", MessageSourceSystem, snapshot.History[1].Source)
	}
}

func TestSessionToSnapshotReturnsDeepCopy(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := NewSession("session-1", ChannelWeb, now)
	session.AppendUserMessage(SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))

	snapshot := session.ToSnapshot()
	snapshot.History[0].Message.Content = "mutated"

	if session.History[0].Message.Content != "hello" {
		t.Fatalf("expected session history to remain unchanged, got %q", session.History[0].Message.Content)
	}
}

func TestRestoreSessionRestoresIndependentSession(t *testing.T) {
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	original := NewSession("session-1", ChannelBot, now)
	original.AppendUserMessage(SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	snapshot := original.ToSnapshot()

	restored := RestoreSession(snapshot)
	restored.AppendAgentMessage(SessionUser{ID: "agent-1", Name: "DM Agent"}, "reply", now.Add(2*time.Minute))
	if restored.Channel != ChannelBot {
		t.Fatalf("expected restored channel %q, got %q", ChannelBot, restored.Channel)
	}

	if len(restored.History) != 2 {
		t.Fatalf("expected restored history length 2, got %d", len(restored.History))
	}
	if len(snapshot.History) != 1 {
		t.Fatalf("expected snapshot history length 1, got %d", len(snapshot.History))
	}
	if len(original.History) != 1 {
		t.Fatalf("expected original history length 1, got %d", len(original.History))
	}
}
