package prompt

import (
	"strings"
	"testing"
	"time"

	agentcontext "DND-AI-BOT/internal/agent/context"
	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
)

func TestComposePreloadedSessionContextPromptRendersRecentMessagesDraftAndMemory(t *testing.T) {
	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	agentCtx := agentcontext.AgentContext{
		SessionID: "session-1",
		Channel:   model.ChannelWeb,
		RecentRecords: []model.HistoryRecord{
			{
				Sequence: 1,
				Source:   model.MessageSourceUser,
				User:     model.SessionUser{ID: "user-1", Name: "Qingke"},
				Message:  model.Message{Content: "创建人类战士"},
			},
			{
				Sequence: 2,
				Source:   model.MessageSourceUser,
				User:     model.SessionUser{ID: "user-1", Name: "Qingke"},
				Message:  model.Message{Content: "Qingke，1级，属性值使用标准点数进行分配"},
			},
		},
	}
	gameState := state.NewGameState("state-1", "session-1", state.PlayerState{
		Draft: &state.CharacterDraft{
			Name:          "Qingke",
			Race:          "人类",
			Class:         "战士",
			Level:         1,
			AbilityMethod: "标准点数",
			PendingFields: []string{"ability_scores"},
		},
	}, now)
	memory := &model.SessionMemory{
		CharacterSummary: "角色创建中：人类战士。",
		SceneSummary:     "准备从 the city 起始场景开始。",
		CurrentObjective: "完成角色卡。",
		RecentKeyEvents:  []string{"用户选择使用标准点数"},
	}
	encounter := combat.NewEncounter("encounter-1", "session-1", []combat.Combatant{
		combat.NewCombatant("hero-1", "Qingke", combat.CombatSideParty, 13, 16, 12),
		combat.NewCombatant("scrap-1", "扭曲的拾荒者", combat.CombatSideEnemy, 22, 12, 10),
	}, now)

	result := ComposePreloadedSessionContextPrompt(agentCtx, gameState, encounter, memory)

	for _, expected := range []string{
		"自动预加载会话上下文",
		"必须优先继承这些事实",
		"创建人类战士",
		"Qingke，1级，属性值使用标准点数进行分配",
		"角色创建草稿",
		"name=Qingke",
		"race=人类",
		"class=战士",
		"ability_method=标准点数",
		"pending_fields=ability_scores",
		"encounter_state",
		"round=1",
		"turn_index=0",
		"当前行动单位=Qingke",
		"扭曲的拾荒者",
		"hp=22/22",
		"ac=12",
		"当前会话长期记忆",
		"完成角色卡",
	} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in preloaded session context prompt, got %q", expected, result)
		}
	}
}

func TestComposePreloadedSessionContextPromptReturnsEmptyWhenNoContext(t *testing.T) {
	result := ComposePreloadedSessionContextPrompt(agentcontext.AgentContext{}, nil, nil, nil)
	if result != "" {
		t.Fatalf("expected empty prompt without context, got %q", result)
	}
}
