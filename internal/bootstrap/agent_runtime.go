package bootstrap

import (
	"errors"
	"log/slog"

	"DND-AI-BOT/internal/agent/client"
	agentcontext "DND-AI-BOT/internal/agent/context"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/agent/tools"
	"DND-AI-BOT/internal/game/rules"
	"DND-AI-BOT/internal/observability"
	retrievalsearch "DND-AI-BOT/internal/retrieval/search"
	"DND-AI-BOT/internal/service"
)

var (
	// ErrInvalidAgentRuntimeInput 表示组装 Runtime 所需的依赖不完整。
	ErrInvalidAgentRuntimeInput = errors.New("invalid agent runtime input")
)

// AgentRuntimeDependencies 承载已经组装完成的 Agent Runtime 依赖。
type AgentRuntimeDependencies struct {
	ModelAdapter client.ModelAdapter
	Registry     tools.Registry
	Executor     tools.Executor
	Runtime      *agentruntime.Runtime
	Config       client.Config
}

// AgentRuntimeInput 定义组装 Agent Runtime 所需的最小业务依赖。
type AgentRuntimeInput struct {
	ContextProvider  agentcontext.Provider
	GameStateService *service.GameStateService
	EncounterService *service.EncounterService
	RuleEngine       rules.RuleEngine
	RuleSearcher     retrievalsearch.Searcher
	LoreSearcher     retrievalsearch.Searcher
	Metrics          *observability.Metrics
	Logger           *slog.Logger
}

// BuildAgentRuntime 将模型层、工具层和 Runtime 组装为一套可运行的 Agent 内核。
func BuildAgentRuntime(input AgentRuntimeInput) (*AgentRuntimeDependencies, error) {
	if err := validateAgentRuntimeInput(input); err != nil {
		return nil, err
	}

	modelAdapter, config, err := BuildModelAdapterFromEnvForRole(client.ModelRolePrimary)
	if err != nil {
		return nil, err
	}
	modelAdapter = client.NewObservedModelAdapter(modelAdapter, config, input.Metrics)

	ruleSearcher := input.RuleSearcher
	loreSearcher := input.LoreSearcher
	if ruleSearcher == nil || loreSearcher == nil {
		searchRuntime, err := BuildSearchRuntime()
		if err != nil {
			return nil, err
		}
		if ruleSearcher == nil {
			ruleSearcher = searchRuntime.RuleSearcher
		}
		if loreSearcher == nil {
			loreSearcher = searchRuntime.LoreSearcher
		}
	}

	toolRuntime, err := BuildToolRuntime(buildToolRuntimeInput(input, ruleSearcher, loreSearcher))
	if err != nil {
		return nil, err
	}

	return &AgentRuntimeDependencies{
		ModelAdapter: modelAdapter,
		Registry:     toolRuntime.Registry,
		Executor:     toolRuntime.Executor,
		Runtime: agentruntime.NewRuntime(
			modelAdapter,
			toolRuntime.Registry,
			toolRuntime.Executor,
			agentruntime.WithRuntimeMetrics(input.Metrics),
			agentruntime.WithRuntimeLogger(input.Logger),
			agentruntime.WithRuntimeModelCallLogConfig(LoadRuntimeObservabilityConfigFromEnv().ModelCallLogConfig),
		),
		Config: config,
	}, nil
}

func validateAgentRuntimeInput(input AgentRuntimeInput) error {
	if input.ContextProvider == nil || input.GameStateService == nil || input.EncounterService == nil || input.RuleEngine == nil {
		return ErrInvalidAgentRuntimeInput
	}

	return nil
}

func buildToolRuntimeInput(
	input AgentRuntimeInput,
	ruleSearcher retrievalsearch.Searcher,
	loreSearcher retrievalsearch.Searcher,
) ToolRuntimeInput {
	return ToolRuntimeInput{
		ContextProvider:  input.ContextProvider,
		GameStateService: input.GameStateService,
		EncounterService: input.EncounterService,
		RuleEngine:       input.RuleEngine,
		RuleSearcher:     ruleSearcher,
		LoreSearcher:     loreSearcher,
		Metrics:          input.Metrics,
		Logger:           input.Logger,
	}
}
