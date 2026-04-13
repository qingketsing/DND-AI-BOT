package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestPGSessionMemoryStoreRoundTrip(t *testing.T) {
	state := newFakePGState()
	state.gameSessions["session-1"] = *model.NewSession("session-1", "user-1", model.ChannelWeb, time.Date(2026, 4, 13, 11, 0, 0, 0, time.UTC))
	db := newFakePGDB(t, state)
	store := NewPGSessionMemoryStore(db)
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	memory := &model.SessionMemory{
		SessionID:        "session-1",
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "the city 广场",
		CurrentObjective: "接取下水道任务",
		RecentKeyEvents:  []string{"创建角色"},
		UpdatedAt:        now,
	}

	if err := store.SaveSessionMemory(context.Background(), memory); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	got, err := store.GetSessionMemoryBySessionID(context.Background(), memory.SessionID)
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.CharacterSummary != memory.CharacterSummary || got.SceneSummary != memory.SceneSummary {
		t.Fatalf("unexpected memory round trip: %+v", got)
	}
	if len(got.RecentKeyEvents) != 1 || got.RecentKeyEvents[0] != "创建角色" {
		t.Fatalf("unexpected memory events: %+v", got.RecentKeyEvents)
	}
}

func TestPGSessionMemoryStoreReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGSessionMemoryStore(db)

	_, err := store.GetSessionMemoryBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrSessionMemoryNotFound) {
		t.Fatalf("expected ErrSessionMemoryNotFound, got %v", err)
	}
}
