package composite

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
)

func TestEncounterRepositoryLoadReturnsCachedValue(t *testing.T) {
	encounter := newTestEncounter()
	cache := &fakeEncounterCache{encounter: encounter}
	store := &fakeEncounterStore{}
	repo := NewCompositeEncounterRepository(store, cache, CachePolicy{})

	got, err := repo.LoadBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.ID != encounter.ID {
		t.Fatalf("expected encounter id %q, got %q", encounter.ID, got.ID)
	}
	if store.getCalls != 0 {
		t.Fatalf("expected store not to be called, got %d", store.getCalls)
	}
}

func TestEncounterRepositoryLoadFallsBackToStoreAndBackfillsCache(t *testing.T) {
	encounter := newTestEncounter()
	cache := &fakeEncounterCache{getErr: repository.ErrCacheMiss}
	store := &fakeEncounterStore{encounter: encounter}
	repo := NewCompositeEncounterRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	got, err := repo.LoadBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.ID != encounter.ID {
		t.Fatalf("expected encounter id %q, got %q", encounter.ID, got.ID)
	}
	if store.getCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.getCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache set to be called once, got %d", cache.setCalls)
	}
}

func TestEncounterRepositoryLoadReturnsNotFoundForNotFoundMarker(t *testing.T) {
	cache := &fakeEncounterCache{getErr: repository.ErrCacheNotFoundMarker}
	store := &fakeEncounterStore{}
	repo := NewCompositeEncounterRepository(store, cache, CachePolicy{})

	_, err := repo.LoadBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrEncounterNotFound) {
		t.Fatalf("expected ErrEncounterNotFound, got %v", err)
	}
}

func TestEncounterRepositoryLoadStoresNotFoundMarkerWhenStoreMisses(t *testing.T) {
	cache := &fakeEncounterCache{getErr: repository.ErrCacheMiss}
	store := &fakeEncounterStore{getErr: repository.ErrEncounterNotFound}
	repo := NewCompositeEncounterRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	_, err := repo.LoadBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrEncounterNotFound) {
		t.Fatalf("expected ErrEncounterNotFound, got %v", err)
	}
	if cache.setNotFoundCalls != 1 {
		t.Fatalf("expected set not found to be called once, got %d", cache.setNotFoundCalls)
	}
}

func TestEncounterRepositorySaveWritesStoreAndDeletesCache(t *testing.T) {
	encounter := newTestEncounter()
	cache := &fakeEncounterCache{}
	store := &fakeEncounterStore{}
	repo := NewCompositeEncounterRepository(store, cache, CachePolicy{})

	if err := repo.Save(context.Background(), encounter); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if store.upsertCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.upsertCalls)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected cache delete to be called once, got %d", cache.deleteCalls)
	}
}

func newTestEncounter() *combat.Encounter {
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	combatants := []combat.Combatant{
		combat.NewCombatant("hero-1", "Hero", combat.CombatSideParty, 20, 15, 12),
		combat.NewCombatant("goblin-1", "Goblin", combat.CombatSideEnemy, 8, 13, 10),
	}
	encounter := combat.NewEncounter("encounter-1", "session-1", combatants, now)
	combatant, _ := encounter.FindCombatant("hero-1")
	combatant.AddEffect(combat.StatusEffect{ID: "effect-1", Type: combat.EffectStunned, Source: "spell", Duration: 1})
	return encounter
}

type fakeEncounterStore struct {
	encounter   *combat.Encounter
	getErr      error
	upsertErr   error
	getCalls    int
	upsertCalls int
}

func (f *fakeEncounterStore) UpsertEncounter(ctx context.Context, encounter *combat.Encounter) error {
	f.upsertCalls++
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.encounter = encounter
	return nil
}

func (f *fakeEncounterStore) GetEncounterBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.encounter, nil
}

type fakeEncounterCache struct {
	encounter        *combat.Encounter
	getErr           error
	setErr           error
	setNotFoundErr   error
	deleteErr        error
	setCalls         int
	setNotFoundCalls int
	deleteCalls      int
}

func (f *fakeEncounterCache) GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.encounter, nil
}

func (f *fakeEncounterCache) Set(ctx context.Context, encounter *combat.Encounter, ttl time.Duration) error {
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.encounter = encounter
	return nil
}

func (f *fakeEncounterCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	f.setNotFoundCalls++
	return f.setNotFoundErr
}

func (f *fakeEncounterCache) DeleteBySessionID(ctx context.Context, sessionID string) error {
	f.deleteCalls++
	return f.deleteErr
}
