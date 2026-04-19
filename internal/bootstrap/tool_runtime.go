package bootstrap

import (
	"errors"

	agentcontext "DND-AI-BOT/internal/agent/context"
	"DND-AI-BOT/internal/agent/tools"
	"DND-AI-BOT/internal/game/rules"
	"DND-AI-BOT/internal/observability"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
	"DND-AI-BOT/internal/service"
)

var (
	// ErrInvalidToolRuntimeInput 表示构建工具运行时所需的依赖不完整。
	ErrInvalidToolRuntimeInput = errors.New("invalid tool runtime input")
)

// ToolRuntimeDependencies 承载已装配好的工具注册表与执行器。
type ToolRuntimeDependencies struct {
	Registry tools.Registry
	Executor tools.Executor
}

// ToolRuntimeInput 定义构建工具运行时所需的最小业务依赖。
type ToolRuntimeInput struct {
	ContextProvider  agentcontext.Provider
	GameStateService *service.GameStateService
	EncounterService *service.EncounterService
	RuleEngine       rules.RuleEngine
	RuleSearcher     retrievalsearch.Searcher
	LoreSearcher     retrievalsearch.Searcher
	Metrics          *observability.Metrics
}

// BuildToolRuntime 根据现有业务依赖构建默认工具注册表与执行器。
func BuildToolRuntime(input ToolRuntimeInput) (*ToolRuntimeDependencies, error) {
	if err := validateToolRuntimeInput(input); err != nil {
		return nil, err
	}

	registry := tools.NewInMemoryRegistry()
	if err := tools.RegisterDefaultTools(registry, buildRegisterDependencies(input)); err != nil {
		return nil, err
	}

	return &ToolRuntimeDependencies{
		Registry: registry,
		Executor: tools.NewExecutor(registry, tools.WithExecutorMetrics(input.Metrics)),
	}, nil
}

func validateToolRuntimeInput(input ToolRuntimeInput) error {
	if input.ContextProvider == nil || input.GameStateService == nil || input.EncounterService == nil || input.RuleEngine == nil {
		return ErrInvalidToolRuntimeInput
	}

	return nil
}

func buildRegisterDependencies(input ToolRuntimeInput) tools.RegisterDependencies {
	return tools.RegisterDependencies{
		ContextProvider:  input.ContextProvider,
		GameStateService: input.GameStateService,
		EncounterService: input.EncounterService,
		RuleEngine:       input.RuleEngine,
		RuleSearcher:     input.RuleSearcher,
		LoreSearcher:     input.LoreSearcher,
		Metrics:          input.Metrics,
	}
}
