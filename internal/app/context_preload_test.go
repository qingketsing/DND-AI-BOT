package app

import (
	"context"
	"strings"
	"testing"
	"time"

	agentcontext "DND-AI-BOT/internal/agent/context"
	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestBuildPreloadedContextPromptLoadsRecentMessagesGameStateAndSessionMemory(t *testing.T) {
	ctx := context.Background()
	gameState := state.NewGameState("state-1", "session-1", state.PlayerState{
		Draft: &state.CharacterDraft{
			Name:          "Qingke",
			Race:          "人类",
			Class:         "战士",
			Level:         1,
			AbilityMethod: "标准点数",
			PendingFields: []string{"ability_scores"},
		},
	}, time.Now())
	provider := &fakePreloadContextProvider{
		result: agentcontext.AgentContext{
			SessionID: "session-1",
			Channel:   model.ChannelWeb,
			RecentRecords: []model.HistoryRecord{
				{
					Sequence: 1,
					Source:   model.MessageSourceUser,
					User:     model.SessionUser{ID: "user-1", Name: "Qingke"},
					Message:  model.Message{Content: "创建人类战士"},
				},
			},
		},
	}
	gameStates := &fakePreloadGameStateReader{state: gameState}
	encounter := combat.NewEncounter("encounter-1", "session-1", []combat.Combatant{
		combat.NewCombatant("hero-1", "Qingke", combat.CombatSideParty, 13, 16, 12),
		combat.NewCombatant("scrap-1", "扭曲的拾荒者", combat.CombatSideEnemy, 22, 12, 10),
	}, time.Now())
	encounters := &fakePreloadEncounterReader{encounter: encounter}
	memories := &fakePreloadSessionMemoryReader{memory: &model.SessionMemory{
		CurrentObjective: "完成角色卡",
	}}

	result, err := buildPreloadedContextPrompt(ctx, preloadedContextInput{
		SessionID:           "session-1",
		ContextLimit:        0,
		ContextProvider:     provider,
		GameStateReader:     gameStates,
		EncounterReader:     encounters,
		SessionMemoryReader: memories,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider.limit != defaultPreloadedContextLimit {
		t.Fatalf("expected default context limit %d, got %d", defaultPreloadedContextLimit, provider.limit)
	}
	for _, expected := range []string{"创建人类战士", "角色创建草稿", "name=Qingke", "encounter_state", "当前行动单位=Qingke", "完成角色卡"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %q in preloaded context prompt, got %q", expected, result)
		}
	}
}

func TestBuildPreloadedContextPromptIgnoresMissingGameState(t *testing.T) {
	result, err := buildPreloadedContextPrompt(context.Background(), preloadedContextInput{
		SessionID: "session-1",
		ContextProvider: &fakePreloadContextProvider{
			result: agentcontext.AgentContext{
				SessionID: "session-1",
				RecentRecords: []model.HistoryRecord{
					{Sequence: 1, Source: model.MessageSourceUser, Message: model.Message{Content: "使用标准数组"}},
				},
			},
		},
		GameStateReader:     &fakePreloadGameStateReader{err: repository.ErrGameStateNotFound},
		EncounterReader:     &fakePreloadEncounterReader{err: repository.ErrEncounterNotFound},
		SessionMemoryReader: &fakePreloadSessionMemoryReader{memory: &model.SessionMemory{}},
	})
	if err != nil {
		t.Fatalf("expected missing game state to be ignored, got %v", err)
	}
	if !strings.Contains(result, "使用标准数组") {
		t.Fatalf("expected recent message in prompt, got %q", result)
	}
}

type fakePreloadContextProvider struct {
	result agentcontext.AgentContext
	limit  int
	err    error
}

func (f *fakePreloadContextProvider) BuildContext(ctx context.Context, sessionID string, limit int) (agentcontext.AgentContext, error) {
	f.limit = limit
	return f.result, f.err
}

type fakePreloadGameStateReader struct {
	state *state.GameState
	err   error
}

func (f *fakePreloadGameStateReader) GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	return f.state, f.err
}

type fakePreloadEncounterReader struct {
	encounter *combat.Encounter
	err       error
}

func (f *fakePreloadEncounterReader) GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	return f.encounter, f.err
}

type fakePreloadSessionMemoryReader struct {
	memory *model.SessionMemory
	err    error
}

func (f *fakePreloadSessionMemoryReader) GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	return f.memory, f.err
}
