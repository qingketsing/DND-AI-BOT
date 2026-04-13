package model_test

import (
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
)

func TestSessionMemoryStoresCoreFields(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	memory := model.SessionMemory{
		SessionID:        "session-1",
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "位于 the city 的广场。",
		CurrentObjective: "寻找守卫队长格伦。",
		RecentKeyEvents:  []string{"创建角色", "查看布告栏"},
		UpdatedAt:        now,
	}

	if memory.SessionID != "session-1" || memory.CharacterSummary == "" || memory.SceneSummary == "" {
		t.Fatalf("expected session memory core fields to be preserved, got %+v", memory)
	}
	if len(memory.RecentKeyEvents) != 2 {
		t.Fatalf("expected recent events to be preserved, got %+v", memory.RecentKeyEvents)
	}
}
