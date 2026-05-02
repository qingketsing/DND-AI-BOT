package bootstrap

import (
	"testing"

	"DND-AI-BOT/internal/agent/client"
)

func TestBuildAgentRuntimeBuildsRuntimeDependencies(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	input := newToolRuntimeInput(t)
	deps, err := BuildAgentRuntime(AgentRuntimeInput{
		ContextProvider:  input.ContextProvider,
		GameStateService: input.GameStateService,
		EncounterService: input.EncounterService,
		RuleEngine:       input.RuleEngine,
	})
	if err != nil {
		t.Fatalf("expected agent runtime build to succeed, got %v", err)
	}
	if deps == nil {
		t.Fatal("expected agent runtime dependencies to be created")
	}
	if deps.ModelAdapter == nil {
		t.Fatal("expected model adapter to be created")
	}
	if deps.Registry == nil {
		t.Fatal("expected registry to be created")
	}
	if deps.Executor == nil {
		t.Fatal("expected executor to be created")
	}
	if deps.Runtime == nil {
		t.Fatal("expected runtime to be created")
	}
	if deps.FastModelAdapter == nil {
		t.Fatal("expected fast model adapter to be created")
	}
	if deps.FastRuntime == nil {
		t.Fatal("expected fast runtime to be created")
	}
	if deps.Config.Provider != client.ProviderMock {
		t.Fatalf("expected provider %q, got %q", client.ProviderMock, deps.Config.Provider)
	}
	if deps.FastConfig.Provider != client.ProviderMock {
		t.Fatalf("expected fast provider %q, got %q", client.ProviderMock, deps.FastConfig.Provider)
	}
}

func TestBuildAgentRuntimeRejectsNilDependencies(t *testing.T) {
	_, err := BuildAgentRuntime(AgentRuntimeInput{})
	if err != ErrInvalidAgentRuntimeInput {
		t.Fatalf("expected ErrInvalidAgentRuntimeInput, got %v", err)
	}
}

func TestBuildAgentRuntimeBuildsRuntimeWithRegisteredTools(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "mock")
	t.Setenv("MODEL_NAME", "")
	t.Setenv("MODEL_API_KEY", "")
	t.Setenv("MODEL_BASE_URL", "")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "")

	input := newToolRuntimeInput(t)
	deps, err := BuildAgentRuntime(AgentRuntimeInput{
		ContextProvider:  input.ContextProvider,
		GameStateService: input.GameStateService,
		EncounterService: input.EncounterService,
		RuleEngine:       input.RuleEngine,
	})
	if err != nil {
		t.Fatalf("expected agent runtime build to succeed, got %v", err)
	}

	specs := deps.Registry.List()
	if len(specs) == 0 {
		t.Fatal("expected registered tool specs to be available")
	}

	names := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		names[spec.Name] = struct{}{}
	}

	for _, required := range []string{"get_game_state", "apply_damage", "resolve_attack_action", "skill_check", "search_rules", "search_lore"} {
		if _, ok := names[required]; !ok {
			t.Fatalf("expected tool %q to be registered in runtime", required)
		}
	}
}

func TestBuildAgentRuntimeUsesModelConfigFromEnv(t *testing.T) {
	t.Setenv("MODEL_PROVIDER", "deepseek")
	t.Setenv("MODEL_NAME", "deepseek-chat")
	t.Setenv("MODEL_API_KEY", "secret")
	t.Setenv("MODEL_BASE_URL", "https://api.deepseek.com")
	t.Setenv("MODEL_TIMEOUT_SECONDS", "30")

	input := newToolRuntimeInput(t)
	deps, err := BuildAgentRuntime(AgentRuntimeInput{
		ContextProvider:  input.ContextProvider,
		GameStateService: input.GameStateService,
		EncounterService: input.EncounterService,
		RuleEngine:       input.RuleEngine,
	})
	if err != nil {
		t.Fatalf("expected deepseek runtime build to succeed, got %v", err)
	}

	if deps.Config.Provider != client.ProviderDeepSeek {
		t.Fatalf("expected provider %q, got %q", client.ProviderDeepSeek, deps.Config.Provider)
	}
	if deps.Config.Model != "deepseek-chat" {
		t.Fatalf("expected model %q, got %q", "deepseek-chat", deps.Config.Model)
	}
	if deps.Config.TimeoutSeconds != 30 {
		t.Fatalf("expected timeout 30, got %d", deps.Config.TimeoutSeconds)
	}
}
