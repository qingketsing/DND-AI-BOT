package service

import (
	"context"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
)

func TestSessionMemoryEventServiceRecordObjectiveChangedUpdatesObjectiveAndEvent(t *testing.T) {
	repo := &fakeSessionMemoryRepository{}
	service := NewSessionMemoryEventService(NewSessionMemoryService(repo))
	now := time.Date(2026, 4, 13, 17, 55, 0, 0, time.UTC)

	if err := service.RecordObjectiveChanged(context.Background(), "session-1", "去市政厅找格伦", "已确认联系人是格伦。", now); err != nil {
		t.Fatalf("expected record objective changed to succeed, got %v", err)
	}

	got := repo.saved["session-1"]
	if got == nil {
		t.Fatal("expected memory to be saved")
	}
	if got.CurrentObjective != "去市政厅找格伦" {
		t.Fatalf("unexpected current objective %+v", got)
	}
	if len(got.RecentKeyEvents) != 1 || got.RecentKeyEvents[0] != "目标更新：已确认联系人是格伦。" {
		t.Fatalf("unexpected recent events %+v", got.RecentKeyEvents)
	}
}

func TestSessionMemoryEventServiceRecordSceneFactUpdatesSceneSummaryAndAppendsFact(t *testing.T) {
	repo := &fakeSessionMemoryRepository{}
	service := NewSessionMemoryEventService(NewSessionMemoryService(repo))
	now := time.Date(2026, 4, 13, 17, 58, 0, 0, time.UTC)

	if err := service.RecordSceneFact(context.Background(), "session-1", "the city 广场", "布告栏显示联系人为格伦。", now); err != nil {
		t.Fatalf("expected record scene fact to succeed, got %v", err)
	}

	got := repo.saved["session-1"]
	if got == nil {
		t.Fatal("expected memory to be saved")
	}
	if got.SceneSummary != "the city 广场" {
		t.Fatalf("unexpected scene summary %+v", got)
	}
	if len(got.RecentKeyEvents) != 1 || got.RecentKeyEvents[0] != "场景事实：布告栏显示联系人为格伦。" {
		t.Fatalf("unexpected recent events %+v", got.RecentKeyEvents)
	}
}

func TestSessionMemoryEventServiceRecordQuestCreatedUpdatesObjectiveAndEvent(t *testing.T) {
	repo := &fakeSessionMemoryRepository{}
	service := NewSessionMemoryEventService(NewSessionMemoryService(repo))
	now := time.Date(2026, 4, 13, 18, 0, 0, 0, time.UTC)

	if err := service.RecordQuestCreated(context.Background(), "session-1", "清理下水道鼠群", "联系人：格伦。", now); err != nil {
		t.Fatalf("expected record quest created to succeed, got %v", err)
	}

	got := repo.saved["session-1"]
	if got == nil {
		t.Fatal("expected memory to be saved")
	}
	if got.CurrentObjective != "联系人：格伦。" {
		t.Fatalf("expected current objective to come from summary, got %+v", got)
	}
	if len(got.RecentKeyEvents) != 1 || got.RecentKeyEvents[0] != "目标更新：已接任务：清理下水道鼠群。 联系人：格伦。" {
		t.Fatalf("unexpected recent events %+v", got.RecentKeyEvents)
	}
}

func TestSessionMemoryEventServiceRecordQuestCompletedSetsWaitingObjective(t *testing.T) {
	repo := &fakeSessionMemoryRepository{
		bySessionID: map[string]*model.SessionMemory{
			"session-1": {
				SessionID:        "session-1",
				CurrentObjective: "去市政厅找格伦",
				RecentKeyEvents:  []string{"已接任务：清理下水道鼠群。"},
			},
		},
	}
	service := NewSessionMemoryEventService(NewSessionMemoryService(repo))
	now := time.Date(2026, 4, 13, 18, 5, 0, 0, time.UTC)

	if err := service.RecordQuestCompleted(context.Background(), "session-1", "清理下水道鼠群", "鼠群已被清除。", now); err != nil {
		t.Fatalf("expected record quest completed to succeed, got %v", err)
	}

	got := repo.saved["session-1"]
	if got == nil {
		t.Fatal("expected memory to be saved")
	}
	if got.CurrentObjective != "等待下一步行动" {
		t.Fatalf("expected waiting objective, got %+v", got)
	}
	if got.RecentKeyEvents[len(got.RecentKeyEvents)-1] != "目标更新：任务完成：清理下水道鼠群。 鼠群已被清除。" {
		t.Fatalf("unexpected recent events %+v", got.RecentKeyEvents)
	}
}
