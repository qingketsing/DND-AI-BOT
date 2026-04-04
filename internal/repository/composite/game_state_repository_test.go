package composite

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
)

func TestGameStateRepositoryLoadReturnsCachedValue(t *testing.T) {
	gameState := newTestGameState()
	cache := &fakeGameStateCache{state: gameState}
	store := &fakeGameStateStore{}
	repo := NewCompositeGameStateRepository(store, cache, CachePolicy{})

	got, err := repo.LoadBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.ID != gameState.ID {
		t.Fatalf("expected game state id %q, got %q", gameState.ID, got.ID)
	}
	if store.getCalls != 0 {
		t.Fatalf("expected store not to be called, got %d", store.getCalls)
	}
}

func TestGameStateRepositoryLoadFallsBackToStoreAndBackfillsCache(t *testing.T) {
	gameState := newTestGameState()
	cache := &fakeGameStateCache{getErr: repository.ErrCacheMiss}
	store := &fakeGameStateStore{state: gameState}
	repo := NewCompositeGameStateRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	got, err := repo.LoadBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.ID != gameState.ID {
		t.Fatalf("expected game state id %q, got %q", gameState.ID, got.ID)
	}
	if store.getCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.getCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache set to be called once, got %d", cache.setCalls)
	}
}

func TestGameStateRepositoryLoadReturnsNotFoundForNotFoundMarker(t *testing.T) {
	cache := &fakeGameStateCache{getErr: repository.ErrCacheNotFoundMarker}
	store := &fakeGameStateStore{}
	repo := NewCompositeGameStateRepository(store, cache, CachePolicy{})

	_, err := repo.LoadBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrGameStateNotFound) {
		t.Fatalf("expected ErrGameStateNotFound, got %v", err)
	}
}

func TestGameStateRepositoryLoadStoresNotFoundMarkerWhenStoreMisses(t *testing.T) {
	cache := &fakeGameStateCache{getErr: repository.ErrCacheMiss}
	store := &fakeGameStateStore{getErr: repository.ErrGameStateNotFound}
	repo := NewCompositeGameStateRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	_, err := repo.LoadBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrGameStateNotFound) {
		t.Fatalf("expected ErrGameStateNotFound, got %v", err)
	}
	if cache.setNotFoundCalls != 1 {
		t.Fatalf("expected set not found to be called once, got %d", cache.setNotFoundCalls)
	}
}

func TestGameStateRepositorySaveWritesStoreAndDeletesCache(t *testing.T) {
	gameState := newTestGameState()
	cache := &fakeGameStateCache{}
	store := &fakeGameStateStore{}
	repo := NewCompositeGameStateRepository(store, cache, CachePolicy{})

	if err := repo.Save(context.Background(), gameState); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if store.upsertCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.upsertCalls)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected cache delete to be called once, got %d", cache.deleteCalls)
	}
}

func newTestGameState() *state.GameState {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	player := state.PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
		Inventory: []state.InventoryItem{
			{ID: "inv-1", ItemID: "potion", Name: "Potion", Quantity: 2},
		},
		Quests: []state.QuestProgress{
			{ID: "quest-1", Title: "Find Key", Status: state.QuestStatusActive},
		},
	}
	gameState := state.NewGameState("state-1", "session-1", player, now)
	gameState.SetCurrentScene("tavern", now.Add(time.Minute))
	return gameState
}

type fakeGameStateStore struct {
	state       *state.GameState
	getErr      error
	upsertErr   error
	getCalls    int
	upsertCalls int
}

func (f *fakeGameStateStore) UpsertGameState(ctx context.Context, state *state.GameState) error {
	f.upsertCalls++
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.state = state
	return nil
}

func (f *fakeGameStateStore) GetGameStateBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.state, nil
}

type fakeGameStateCache struct {
	state            *state.GameState
	getErr           error
	setErr           error
	setNotFoundErr   error
	deleteErr        error
	setCalls         int
	setNotFoundCalls int
	deleteCalls      int
}

func (f *fakeGameStateCache) GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.state, nil
}

func (f *fakeGameStateCache) Set(ctx context.Context, state *state.GameState, ttl time.Duration) error {
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.state = state
	return nil
}

func (f *fakeGameStateCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	f.setNotFoundCalls++
	return f.setNotFoundErr
}

func (f *fakeGameStateCache) DeleteBySessionID(ctx context.Context, sessionID string) error {
	f.deleteCalls++
	return f.deleteErr
}
