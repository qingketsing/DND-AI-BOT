package tools

import (
	agentcontext "DND-AI-BOT/internal/agent/context"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
)

// RegisterDependencies 定义默认工具注册所需的全部依赖。
type RegisterDependencies struct {
	ContextProvider  agentcontext.Provider
	GameStateService gameStateToolService
	EncounterService encounterToolService
	RuleEngine       ruleToolEngine
	RuleSearcher     retrievalsearch.Searcher
	LoreSearcher     retrievalsearch.Searcher
}

// RegisterDefaultTools 将当前默认工具集合一次性注册到注册表中。
func RegisterDefaultTools(registry Registry, deps RegisterDependencies) error {
	tools := []Tool{
		NewGetAgentContextTool(deps.ContextProvider),
		NewGetGameStateTool(deps.GameStateService),
		NewUpdateStatsTool(deps.GameStateService),
		NewAddItemTool(deps.GameStateService),
		NewRemoveItemTool(deps.GameStateService),
		NewAddGoldTool(deps.GameStateService),
		NewSpendGoldTool(deps.GameStateService),
		NewSetSceneTool(deps.GameStateService),
		NewUpsertQuestTool(deps.GameStateService),
		NewGetEncounterTool(deps.EncounterService),
		NewApplyDamageTool(deps.EncounterService),
		NewHealTool(deps.EncounterService),
		NewAdvanceTurnTool(deps.EncounterService),
		NewAddEffectTool(deps.EncounterService),
		NewRemoveEffectTool(deps.EncounterService),
		NewCanActTool(deps.EncounterService),
		NewRollDiceTool(deps.RuleEngine),
		NewAbilityCheckTool(deps.RuleEngine),
		NewSkillCheckTool(deps.RuleEngine),
	}
	if deps.RuleSearcher != nil {
		tools = append(tools, NewSearchRulesTool(deps.RuleSearcher))
	}
	if deps.LoreSearcher != nil {
		tools = append(tools, NewSearchLoreTool(deps.LoreSearcher))
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return err
		}
	}

	return nil
}
