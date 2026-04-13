package prompt

import (
	"strings"
	"testing"

	"DND-AI-BOT/internal/model"
)

func TestComposeSessionMemoryPromptRendersNonEmptySections(t *testing.T) {
	memory := &model.SessionMemory{
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "位于 the city 广场。",
		CurrentObjective: "寻找格伦。",
		RecentKeyEvents:  []string{"创建角色", "查看布告栏"},
	}

	result := ComposeSessionMemoryPrompt(memory)

	for _, expected := range []string{"当前会话长期记忆", "青稞，精灵法师。", "位于 the city 广场。", "寻找格伦。", "查看布告栏"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in session memory prompt, got %q", expected, result)
		}
	}
}

func TestComposeSessionMemoryPromptReturnsEmptyForNilOrEmptyMemory(t *testing.T) {
	if result := ComposeSessionMemoryPrompt(nil); result != "" {
		t.Fatalf("expected empty prompt for nil memory, got %q", result)
	}
	if result := ComposeSessionMemoryPrompt(&model.SessionMemory{}); result != "" {
		t.Fatalf("expected empty prompt for empty memory, got %q", result)
	}
}
