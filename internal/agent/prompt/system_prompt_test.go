package prompt

import (
	"strings"
	"testing"
)

func TestDefaultSystemPromptLimitsSupplementalSearchToOneRetry(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "最多补查 1 次") {
		t.Fatalf("expected prompt to limit supplemental search to one retry, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptRequiresAgentContextForSessionFacts(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "get_agent_context") {
		t.Fatalf("expected prompt to mention get_agent_context for session facts, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptRequiresCreateCharacterForCharacterCreation(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "create_character") {
		t.Fatalf("expected prompt to mention create_character for character creation, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptTreatsWarmupAsBackgroundNotFinalEvidence(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "预热摘要") {
		t.Fatalf("expected prompt to mention warmup summaries, got %q", DefaultSystemPrompt)
	}
	if !strings.Contains(DefaultSystemPrompt, "不代替") {
		t.Fatalf("expected prompt to state warmup does not replace retrieval, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptAvoidsRecreatingExistingCharacter(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "不要重复要求玩家重新创建角色") {
		t.Fatalf("expected prompt to avoid repeated character creation requests, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptResolvesStateConflictWithContextFirst(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "如果结构化状态与当前会话事实看起来冲突") {
		t.Fatalf("expected prompt to mention state conflict resolution, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptRequiresDraftUpdatesForPartialCharacterInfo(t *testing.T) {
	if !strings.Contains(DefaultSystemPrompt, "upsert_character_draft") {
		t.Fatalf("expected prompt to mention upsert_character_draft for partial character info, got %q", DefaultSystemPrompt)
	}
}

func TestDefaultSystemPromptRequiresCreateEncounterBeforeCombatTools(t *testing.T) {
	for _, expected := range []string{
		"create_encounter",
		"apply_damage",
		"不要只用文本宣布战斗开始",
	} {
		if !strings.Contains(DefaultSystemPrompt, expected) {
			t.Fatalf("expected prompt to mention %q for structured combat state, got %q", expected, DefaultSystemPrompt)
		}
	}
}

func TestDefaultSystemPromptForbidsInventingStateValueSources(t *testing.T) {
	for _, expected := range []string{
		"不要编造原因解释该数值",
		"来源未确认",
		"玩家追问“这个数值从哪来”",
	} {
		if !strings.Contains(DefaultSystemPrompt, expected) {
			t.Fatalf("expected prompt to mention %q for state value provenance, got %q", expected, DefaultSystemPrompt)
		}
	}
}
