package bootstrap

import (
	"context"
	"testing"
	"time"

	agentcontext "DND-AI-BOT/internal/agent/context"
	"DND-AI-BOT/internal/agent/tools"
	basecontext "DND-AI-BOT/internal/context"
	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/game/rules"
	gamestate "DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/repository/memory"
	"DND-AI-BOT/internal/service"
)

func TestBuildToolRuntimeBuildsRegistryAndExecutor(t *testing.T) {
	deps := newToolRuntimeInput(t)
	searchDeps, err := BuildSearchRuntime()
	if err != nil {
		t.Fatalf("expected search runtime build to succeed, got %v", err)
	}
	deps.RuleSearcher = searchDeps.RuleSearcher
	deps.LoreSearcher = searchDeps.LoreSearcher

	runtimeDeps, err := BuildToolRuntime(deps)
	if err != nil {
		t.Fatalf("expected tool runtime build to succeed, got %v", err)
	}
	if runtimeDeps == nil {
		t.Fatal("expected tool runtime dependencies to be created")
	}
	if runtimeDeps.Registry == nil {
		t.Fatal("expected registry to be created")
	}
	if runtimeDeps.Executor == nil {
		t.Fatal("expected executor to be created")
	}
}

func TestBuildToolRuntimeRegistersDefaultTools(t *testing.T) {
	deps := newToolRuntimeInput(t)
	searchDeps, err := BuildSearchRuntime()
	if err != nil {
		t.Fatalf("expected search runtime build to succeed, got %v", err)
	}
	deps.RuleSearcher = searchDeps.RuleSearcher
	deps.LoreSearcher = searchDeps.LoreSearcher

	runtimeDeps, err := BuildToolRuntime(deps)
	if err != nil {
		t.Fatalf("expected tool runtime build to succeed, got %v", err)
	}

	specs := runtimeDeps.Registry.List()
	if len(specs) == 0 {
		t.Fatal("expected default tools to be registered")
	}

	names := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		names[spec.Name] = struct{}{}
	}

	for _, required := range []string{"get_game_state", "apply_damage", "skill_check", "search_rules", "search_lore"} {
		if _, ok := names[required]; !ok {
			t.Fatalf("expected tool %q to be registered", required)
		}
	}
}

func TestBuildToolRuntimeRejectsNilDependencies(t *testing.T) {
	_, err := BuildToolRuntime(ToolRuntimeInput{})
	if err != ErrInvalidToolRuntimeInput {
		t.Fatalf("expected ErrInvalidToolRuntimeInput, got %v", err)
	}
}

func TestBuildToolRuntimeExecutorCanExecuteRegisteredTool(t *testing.T) {
	now := time.Now().UTC()
	deps := newToolRuntimeInput(t)
	searchDeps, err := BuildSearchRuntime()
	if err != nil {
		t.Fatalf("expected search runtime build to succeed, got %v", err)
	}
	deps.RuleSearcher = searchDeps.RuleSearcher
	deps.LoreSearcher = searchDeps.LoreSearcher
	runtimeDeps, err := BuildToolRuntime(deps)
	if err != nil {
		t.Fatalf("expected tool runtime build to succeed, got %v", err)
	}

	output, err := runtimeDeps.Executor.Execute(context.Background(), "get_game_state", newToolCallInput("session-1", "{}", now))
	if err != nil {
		t.Fatalf("expected executor to run registered tool, got %v", err)
	}
	if output.ToolName != "get_game_state" {
		t.Fatalf("expected tool name %q, got %q", "get_game_state", output.ToolName)
	}
	if output.Content == nil {
		t.Fatal("expected tool output content to be populated")
	}
}

func newToolRuntimeInput(t *testing.T) ToolRuntimeInput {
	t.Helper()

	now := time.Now().UTC()
	sessionRepo := memory.NewSessionRepository()
	session := model.NewSession("session-1", model.ChannelBot, now)
	session.AppendUserMessage(model.User{ID: "user-1", Name: "Hero"}, "hello", now)
	if err := sessionRepo.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	contextStore := basecontext.NewSessionContextStore(sessionRepo)
	contextProvider := agentcontext.NewProvider(contextStore)

	gameStateRepo := newFakeGameStateRepository()
	gameStateService := service.NewGameStateService(gameStateRepo)
	_, err := gameStateService.Create(context.Background(), service.CreateGameStateInput{
		ID:        "game-state-1",
		SessionID: "session-1",
		Player: gamestate.PlayerState{
			Name:  "Hero",
			Level: 1,
			Gold:  10,
			Stats: gamestate.CharacterStats{
				STR: 10,
				DEX: 12,
				CON: 11,
				INT: 13,
				WIS: 9,
				CHA: 8,
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("expected game state create to succeed, got %v", err)
	}

	encounterRepo := newFakeEncounterRepository()
	encounterService := service.NewEncounterService(encounterRepo)
	_, err = encounterService.Create(context.Background(), service.CreateEncounterInput{
		ID:        "encounter-1",
		SessionID: "session-1",
		Combatants: []combat.Combatant{
			combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 12, 14, 2),
			combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 7, 12, 1),
		},
	}, now)
	if err != nil {
		t.Fatalf("expected encounter create to succeed, got %v", err)
	}

	return ToolRuntimeInput{
		ContextProvider:  contextProvider,
		GameStateService: gameStateService,
		EncounterService: encounterService,
		RuleEngine:       rules.NewDefaultRuleEngine(nil),
	}
}

type fakeGameStateRepository struct {
	states map[string]*gamestate.GameState
}

func newFakeGameStateRepository() *fakeGameStateRepository {
	return &fakeGameStateRepository{
		states: make(map[string]*gamestate.GameState),
	}
}

func (r *fakeGameStateRepository) Save(ctx context.Context, state *gamestate.GameState) error {
	_ = ctx
	if state == nil {
		return repository.ErrGameStateNotFound
	}
	r.states[state.SessionID] = state
	return nil
}

func (r *fakeGameStateRepository) LoadBySessionID(ctx context.Context, sessionID string) (*gamestate.GameState, error) {
	_ = ctx
	state, ok := r.states[sessionID]
	if !ok {
		return nil, repository.ErrGameStateNotFound
	}
	return state, nil
}

type fakeEncounterRepository struct {
	encounters map[string]*combat.Encounter
}

func newFakeEncounterRepository() *fakeEncounterRepository {
	return &fakeEncounterRepository{
		encounters: make(map[string]*combat.Encounter),
	}
}

func (r *fakeEncounterRepository) Save(ctx context.Context, encounter *combat.Encounter) error {
	_ = ctx
	if encounter == nil {
		return repository.ErrEncounterNotFound
	}
	r.encounters[encounter.SessionID] = encounter
	return nil
}

func (r *fakeEncounterRepository) LoadBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	_ = ctx
	encounter, ok := r.encounters[sessionID]
	if !ok {
		return nil, repository.ErrEncounterNotFound
	}
	return encounter, nil
}

func newToolCallInput(sessionID string, raw string, now time.Time) tools.CallInput {
	return tools.CallInput{
		SessionID: sessionID,
		Raw:       []byte(raw),
		Now:       now,
	}
}
