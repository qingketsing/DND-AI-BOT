package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestSessionMemoryServiceUpdateAppendsEventAndCapsHistory(t *testing.T) {
	repo := &fakeSessionMemoryRepository{
		bySessionID: map[string]*model.SessionMemory{
			"session-1": {
				SessionID:       "session-1",
				RecentKeyEvents: []string{"e1", "e2", "e3", "e4", "e5"},
			},
		},
	}
	service := NewSessionMemoryService(repo)
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	got, err := service.Update(context.Background(), UpdateSessionMemoryInput{
		SessionID:        "session-1",
		SceneSummary:     "the city 广场",
		CurrentObjective: "寻找格伦",
		AppendEvent:      "查看布告栏",
	}, now)
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if got.SceneSummary != "the city 广场" || got.CurrentObjective != "寻找格伦" {
		t.Fatalf("expected fields to update, got %+v", got)
	}
	if len(got.RecentKeyEvents) != 5 || got.RecentKeyEvents[4] != "查看布告栏" {
		t.Fatalf("expected capped event history, got %+v", got.RecentKeyEvents)
	}
}

func TestSessionMemoryServiceGetReturnsDefaultWhenMissing(t *testing.T) {
	repo := &fakeSessionMemoryRepository{}
	service := NewSessionMemoryService(repo)

	got, err := service.GetBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}
	if got.SessionID != "session-1" {
		t.Fatalf("expected default session id, got %+v", got)
	}
	if len(got.RecentKeyEvents) != 0 {
		t.Fatalf("expected empty default memory, got %+v", got)
	}
}

func TestSessionMemoryServiceMergeSummaryOverwritesSummaries(t *testing.T) {
	repo := &fakeSessionMemoryRepository{
		bySessionID: map[string]*model.SessionMemory{
			"session-1": {
				SessionID:        "session-1",
				CharacterSummary: "旧角色",
				SceneSummary:     "旧场景",
				CurrentObjective: "旧目标",
				RecentKeyEvents:  []string{"旧事件"},
			},
		},
	}
	service := NewSessionMemoryService(repo)
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	got, err := service.MergeSummary(context.Background(), MergeSummaryInput{
		SessionID:        "session-1",
		CharacterSummary: "新角色",
		SceneSummary:     "新场景",
		CurrentObjective: "新目标",
		RecentKeyEvents:  []string{"事件1", "事件2"},
	}, now)
	if err != nil {
		t.Fatalf("expected merge to succeed, got %v", err)
	}
	if got.CharacterSummary != "新角色" || got.SceneSummary != "新场景" || got.CurrentObjective != "新目标" {
		t.Fatalf("expected summary fields to overwrite, got %+v", got)
	}
	if len(got.RecentKeyEvents) != 2 || got.RecentKeyEvents[1] != "事件2" {
		t.Fatalf("expected merged events, got %+v", got.RecentKeyEvents)
	}
}

type fakeSessionMemoryRepository struct {
	bySessionID map[string]*model.SessionMemory
	saved       map[string]*model.SessionMemory
	loadErr     error
	saveErr     error
}

func (f *fakeSessionMemoryRepository) LoadBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	_ = ctx
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	if f.bySessionID == nil {
		return nil, repository.ErrSessionMemoryNotFound
	}
	memory, ok := f.bySessionID[sessionID]
	if !ok {
		return nil, repository.ErrSessionMemoryNotFound
	}
	return cloneSessionMemory(memory), nil
}

func (f *fakeSessionMemoryRepository) Save(ctx context.Context, memory *model.SessionMemory) error {
	_ = ctx
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.bySessionID == nil {
		f.bySessionID = make(map[string]*model.SessionMemory)
	}
	if f.saved == nil {
		f.saved = make(map[string]*model.SessionMemory)
	}
	cloned := cloneSessionMemory(memory)
	f.bySessionID[memory.SessionID] = cloned
	f.saved[memory.SessionID] = cloned
	return nil
}

func cloneSessionMemory(memory *model.SessionMemory) *model.SessionMemory {
	if memory == nil {
		return nil
	}
	cloned := *memory
	cloned.RecentKeyEvents = append([]string(nil), memory.RecentKeyEvents...)
	return &cloned
}

var _ repository.SessionMemoryRepository = (*fakeSessionMemoryRepository)(nil)

func TestSessionMemoryServiceGetPropagatesUnexpectedErrors(t *testing.T) {
	repo := &fakeSessionMemoryRepository{loadErr: errors.New("boom")}
	service := NewSessionMemoryService(repo)

	_, err := service.GetBySessionID(context.Background(), "session-1")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected propagated error, got %v", err)
	}
}
