package bootstrap

import (
	"errors"

	"DND-AI-BOT/internal/agent/client"
	agentcontext "DND-AI-BOT/internal/agent/context"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/agent/tools"
	"DND-AI-BOT/internal/game/rules"
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
}

// BuildAgentRuntime 将模型层、工具层和 Runtime 组装为一套可运行的 Agent 内核。
func BuildAgentRuntime(input AgentRuntimeInput) (*AgentRuntimeDependencies, error) {
	if err := validateAgentRuntimeInput(input); err != nil {
		return nil, err
	}

	modelAdapter, config, err := BuildModelAdapterFromEnv()
	if err != nil {
		return nil, err
	}

	toolRuntime, err := BuildToolRuntime(buildToolRuntimeInput(input))
	if err != nil {
		return nil, err
	}

	return &AgentRuntimeDependencies{
		ModelAdapter: modelAdapter,
		Registry:     toolRuntime.Registry,
		Executor:     toolRuntime.Executor,
		Runtime:      agentruntime.NewRuntime(modelAdapter, toolRuntime.Registry, toolRuntime.Executor),
		Config:       config,
	}, nil
}

func validateAgentRuntimeInput(input AgentRuntimeInput) error {
	if input.ContextProvider == nil || input.GameStateService == nil || input.EncounterService == nil || input.RuleEngine == nil {
		return ErrInvalidAgentRuntimeInput
	}

	return nil
}

func buildToolRuntimeInput(input AgentRuntimeInput) ToolRuntimeInput {
	return ToolRuntimeInput{
		ContextProvider:  input.ContextProvider,
		GameStateService: input.GameStateService,
		EncounterService: input.EncounterService,
		RuleEngine:       input.RuleEngine,
	}
}
