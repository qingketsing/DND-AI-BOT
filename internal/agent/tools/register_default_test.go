package tools

import (
	"testing"

	agentcontext "DND-AI-BOT/internal/agent/context"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

func TestRegisterDefaultToolsRegistersAllPlannedTools(t *testing.T) {
	registry := NewInMemoryRegistry()

	err := RegisterDefaultTools(registry, RegisterDependencies{
		ContextProvider:  &fakeContextProvider{},
		GameStateService: &fakeGameStateToolService{result: newToolGameState()},
		EncounterService: &fakeEncounterToolService{result: newToolEncounter()},
		RuleEngine:       &fakeRuleEngine{},
		RuleSearcher:     &fakeKnowledgeSearcher{},
		LoreSearcher:     &fakeKnowledgeSearcher{},
	})
	if err != nil {
		t.Fatalf("expected register default tools to succeed, got %v", err)
	}

	specs := registry.List()
	if len(specs) != 24 {
		t.Fatalf("expected 24 registered tools, got %d", len(specs))
	}

	expected := []string{
		"ability_check",
		"add_effect",
		"add_gold",
		"add_item",
		"advance_turn",
		"apply_damage",
		"can_act",
		"create_character",
		"create_encounter",
		"get_agent_context",
		"get_encounter",
		"get_game_state",
		"heal",
		"remove_effect",
		"remove_item",
		"roll_dice",
		"search_lore",
		"search_rules",
		"set_scene",
		"skill_check",
		"spend_gold",
		"update_stats",
		"upsert_character_draft",
		"upsert_quest",
	}
	for i, spec := range specs {
		if spec.Name != expected[i] {
			t.Fatalf("expected tool %q at index %d, got %q", expected[i], i, spec.Name)
		}
	}
}

var _ agentcontext.Provider = (*fakeContextProvider)(nil)
var _ gameStateToolService = (*fakeGameStateToolService)(nil)
var _ encounterToolService = (*fakeEncounterToolService)(nil)
var _ ruleToolEngine = (*fakeRuleEngine)(nil)
var _ retrievalsearch.Searcher = (*fakeKnowledgeSearcher)(nil)
