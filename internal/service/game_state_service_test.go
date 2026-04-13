package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
)

func TestCreateGameStateSavesInitialState(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)

	gameState, err := service.Create(ctx, CreateGameStateInput{
		ID:        "state-1",
		SessionID: "session-1",
		Player: state.PlayerState{
			Name:  "Alice",
			Level: 1,
			Gold:  10,
			Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
		},
	}, now)
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if gameState.ID != "state-1" {
		t.Fatalf("expected game state id %q, got %q", "state-1", gameState.ID)
	}
	if repo.saved == nil || repo.saved.SessionID != "session-1" {
		t.Fatal("expected repository to save created game state")
	}
}

func TestGetGameStateReturnsStoredState(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	gameState, err := service.GetBySessionID(ctx, "session-1")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}
	if gameState.ID != "state-1" {
		t.Fatalf("expected game state id %q, got %q", "state-1", gameState.ID)
	}
}

func TestUpdateStatsReplacesPlayerStats(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.UpdateStats(ctx, UpdateStatsInput{
		SessionID: "session-1",
		Stats:     state.CharacterStats{STR: 15, DEX: 14, CON: 13, INT: 12, WIS: 11, CHA: 10},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected update stats to succeed, got %v", err)
	}
	if updated.Player.Stats.STR != 15 || updated.Player.Stats.CHA != 10 {
		t.Fatalf("expected stats to be replaced, got %+v", updated.Player.Stats)
	}
}

func TestCreateCharacterInitializesNewStateWhenMissing(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)

	gameState, err := service.CreateCharacter(ctx, CreateCharacterInput{
		SessionID:         "session-1",
		Name:              "青稞",
		Race:              "精灵",
		Class:             "法师",
		BackgroundSummary: "来自 the city 的年轻法师",
		Level:             1,
		Stats:             state.CharacterStats{STR: 8, DEX: 14, CON: 13, INT: 16, WIS: 12, CHA: 10},
		Inventory: []state.InventoryItem{
			{ID: "inv-1", ItemID: "staff", Name: "法杖", Quantity: 1},
		},
		Scene: "the city 的旅店房间",
	}, now)
	if err != nil {
		t.Fatalf("expected create character to succeed, got %v", err)
	}
	if gameState.ID != "state-session-1" {
		t.Fatalf("expected game state id %q, got %q", "state-session-1", gameState.ID)
	}
	if gameState.Player.Name != "青稞" || gameState.Player.Class != "法师" || gameState.Player.Race != "精灵" {
		t.Fatalf("expected character identity fields to be initialized, got %+v", gameState.Player)
	}
	if gameState.CurrentScene != "the city 的旅店房间" {
		t.Fatalf("expected scene to be initialized, got %q", gameState.CurrentScene)
	}
	if len(gameState.Player.Inventory) != 1 || gameState.Player.Inventory[0].ItemID != "staff" {
		t.Fatalf("expected inventory to be initialized, got %+v", gameState.Player.Inventory)
	}
}

func TestCreateCharacterUpdatesSessionMemory(t *testing.T) {
	gameStates := newFakeGameStateRepository()
	memories := &fakeSessionMemoryRepository{}
	service := NewGameStateService(gameStates, NewSessionMemoryService(memories))
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	_, err := service.CreateCharacter(context.Background(), CreateCharacterInput{
		SessionID: "session-1",
		Name:      "青稞",
		Race:      "精灵",
		Class:     "法师",
		Level:     1,
		Scene:     "the city 广场",
	}, now)
	if err != nil {
		t.Fatalf("expected create character to succeed, got %v", err)
	}

	got, ok := memories.saved["session-1"]
	if !ok {
		t.Fatal("expected session memory to be updated")
	}
	if got.CharacterSummary == "" || got.SceneSummary != "the city 广场" {
		t.Fatalf("unexpected memory update %+v", got)
	}
	if len(got.RecentKeyEvents) == 0 {
		t.Fatalf("expected memory events to be recorded, got %+v", got)
	}
}

func TestUpsertCharacterDraftMergesPartialFieldsAcrossTurns(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 12, 13, 0, 0, 0, time.UTC)

	updated, err := service.UpsertCharacterDraft(ctx, UpsertCharacterDraftInput{
		SessionID: "session-1",
		Name:      "青稞",
		Race:      "精灵",
		Class:     "法师",
	}, now)
	if err != nil {
		t.Fatalf("expected first draft upsert to succeed, got %v", err)
	}
	if updated.Player.Draft == nil || updated.Player.Draft.Name != "青稞" || updated.Player.Draft.Class != "法师" {
		t.Fatalf("expected initial character draft, got %+v", updated.Player.Draft)
	}

	updated, err = service.UpsertCharacterDraft(ctx, UpsertCharacterDraftInput{
		SessionID:      "session-1",
		AbilityMethod:  "standard_array",
		PendingFields:  []string{"level", "stats_assignment"},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected second draft upsert to succeed, got %v", err)
	}
	if updated.Player.Draft == nil {
		t.Fatal("expected draft to remain present")
	}
	if updated.Player.Draft.Name != "青稞" || updated.Player.Draft.Race != "精灵" || updated.Player.Draft.Class != "法师" {
		t.Fatalf("expected previous identity fields to be preserved, got %+v", updated.Player.Draft)
	}
	if updated.Player.Draft.AbilityMethod != "standard_array" {
		t.Fatalf("expected ability method to be updated, got %+v", updated.Player.Draft)
	}
	if len(updated.Player.Draft.PendingFields) != 2 {
		t.Fatalf("expected pending fields to be updated, got %+v", updated.Player.Draft.PendingFields)
	}
}

func TestAddItemUpdatesInventory(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.AddItem(ctx, AddItemInput{
		SessionID: "session-1",
		Item:      state.InventoryItem{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 2},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected add item to succeed, got %v", err)
	}
	if len(updated.Player.Inventory) != 1 || updated.Player.Inventory[0].Quantity != 2 {
		t.Fatalf("expected inventory to contain new item, got %+v", updated.Player.Inventory)
	}
}

func TestRemoveItemReturnsErrorWhenQuantityIsInsufficient(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	player := newTestPlayerState()
	player.Inventory = []state.InventoryItem{{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 1}}
	existing := state.NewGameState("state-1", "session-1", player, now)
	repo.bySessionID["session-1"] = existing

	_, err := service.RemoveItem(ctx, RemoveItemInput{
		SessionID: "session-1",
		ItemID:    "potion",
		Quantity:  2,
	}, now.Add(time.Minute))
	if !errors.Is(err, ErrInsufficientItemQuantity) {
		t.Fatalf("expected ErrInsufficientItemQuantity, got %v", err)
	}
}

func TestAddGoldUpdatesGold(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.AddGold(ctx, AddGoldInput{SessionID: "session-1", Amount: 5}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected add gold to succeed, got %v", err)
	}
	if updated.Player.Gold != 15 {
		t.Fatalf("expected gold 15, got %d", updated.Player.Gold)
	}
}

func TestSpendGoldReturnsErrorWhenGoldIsInsufficient(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	_, err := service.SpendGold(ctx, SpendGoldInput{SessionID: "session-1", Amount: 99}, now.Add(time.Minute))
	if !errors.Is(err, ErrInsufficientGold) {
		t.Fatalf("expected ErrInsufficientGold, got %v", err)
	}
}

func TestSetSceneUpdatesCurrentScene(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.SetScene(ctx, SetSceneInput{SessionID: "session-1", Scene: "forest"}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected set scene to succeed, got %v", err)
	}
	if updated.CurrentScene != "forest" {
		t.Fatalf("expected current scene %q, got %q", "forest", updated.CurrentScene)
	}
}

func TestSetSceneUpdatesSessionMemory(t *testing.T) {
	repo := newFakeGameStateRepository()
	memories := &fakeSessionMemoryRepository{}
	service := NewGameStateService(repo, NewSessionMemoryService(memories))
	ctx := context.Background()
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	_, err := service.SetScene(ctx, SetSceneInput{SessionID: "session-1", Scene: "the city 广场"}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected set scene to succeed, got %v", err)
	}

	got, ok := memories.saved["session-1"]
	if !ok {
		t.Fatal("expected session memory to be updated")
	}
	if got.SceneSummary != "the city 广场" {
		t.Fatalf("unexpected memory scene summary %+v", got)
	}
	if len(got.RecentKeyEvents) == 0 || got.RecentKeyEvents[len(got.RecentKeyEvents)-1] != "场景事实：当前场景已切换为：the city 广场" {
		t.Fatalf("unexpected scene fact events %+v", got.RecentKeyEvents)
	}
}

func TestUpsertQuestAddsAndUpdatesQuest(t *testing.T) {
	repo := newFakeGameStateRepository()
	service := NewGameStateService(repo)
	ctx := context.Background()
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	updated, err := service.UpsertQuest(ctx, UpsertQuestInput{
		SessionID: "session-1",
		Quest:     state.QuestProgress{ID: "quest-1", Title: "Find Key", Status: state.QuestStatusActive},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected upsert quest to succeed, got %v", err)
	}
	if len(updated.Player.Quests) != 1 {
		t.Fatalf("expected 1 quest, got %d", len(updated.Player.Quests))
	}

	updated, err = service.UpsertQuest(ctx, UpsertQuestInput{
		SessionID: "session-1",
		Quest:     state.QuestProgress{ID: "quest-1", Title: "Find Key", Status: state.QuestStatusCompleted},
	}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("expected second upsert quest to succeed, got %v", err)
	}
	if updated.Player.Quests[0].Status != state.QuestStatusCompleted {
		t.Fatalf("expected quest to be updated, got %q", updated.Player.Quests[0].Status)
	}
}

func TestUpsertQuestUpdatesSessionMemoryForQuestLifecycle(t *testing.T) {
	repo := newFakeGameStateRepository()
	memories := &fakeSessionMemoryRepository{}
	service := NewGameStateService(repo, NewSessionMemoryService(memories))
	ctx := context.Background()
	now := time.Date(2026, 4, 13, 19, 0, 0, 0, time.UTC)
	existing := state.NewGameState("state-1", "session-1", newTestPlayerState(), now)
	repo.bySessionID["session-1"] = existing

	_, err := service.UpsertQuest(ctx, UpsertQuestInput{
		SessionID: "session-1",
		Quest: state.QuestProgress{
			ID:          "quest-1",
			Title:       "清理下水道鼠群",
			Status:      state.QuestStatusActive,
			Description: "联系人：格伦。",
		},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected active quest upsert to succeed, got %v", err)
	}

	got := memories.saved["session-1"]
	if got == nil {
		t.Fatal("expected memory to be updated for active quest")
	}
	if got.CurrentObjective != "联系人：格伦。" {
		t.Fatalf("expected current objective to be updated, got %+v", got)
	}
	if len(got.RecentKeyEvents) == 0 || got.RecentKeyEvents[len(got.RecentKeyEvents)-1] != "目标更新：已接任务：清理下水道鼠群。 联系人：格伦。" {
		t.Fatalf("unexpected recent events %+v", got.RecentKeyEvents)
	}

	_, err = service.UpsertQuest(ctx, UpsertQuestInput{
		SessionID: "session-1",
		Quest: state.QuestProgress{
			ID:          "quest-1",
			Title:       "清理下水道鼠群",
			Status:      state.QuestStatusCompleted,
			Description: "鼠群已被清除。",
		},
	}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("expected completed quest upsert to succeed, got %v", err)
	}

	got = memories.saved["session-1"]
	if got.CurrentObjective != "等待下一步行动" {
		t.Fatalf("expected waiting objective after completion, got %+v", got)
	}
	if got.RecentKeyEvents[len(got.RecentKeyEvents)-1] != "目标更新：任务完成：清理下水道鼠群。 鼠群已被清除。" {
		t.Fatalf("unexpected completion event %+v", got.RecentKeyEvents)
	}
}

func newTestPlayerState() state.PlayerState {
	return state.PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
	}
}

type fakeGameStateRepository struct {
	bySessionID map[string]*state.GameState
	saved       *state.GameState
	saveErr     error
	loadErr     error
}

func newFakeGameStateRepository() *fakeGameStateRepository {
	return &fakeGameStateRepository{
		bySessionID: make(map[string]*state.GameState),
	}
}

func (f *fakeGameStateRepository) Save(ctx context.Context, gameState *state.GameState) error {
	_ = ctx
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = gameState
	f.bySessionID[gameState.SessionID] = gameState
	return nil
}

func (f *fakeGameStateRepository) LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	_ = ctx
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	gameState, ok := f.bySessionID[sessionID]
	if !ok {
		return nil, repository.ErrGameStateNotFound
	}
	return gameState, nil
}
