package composite

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestSessionMemoryRepositoryLoadReturnsCachedValue(t *testing.T) {
	memory := newTestSessionMemory()
	cache := &fakeSessionMemoryCache{memory: memory}
	store := &fakeSessionMemoryStore{}
	repo := NewCompositeSessionMemoryRepository(store, cache, CachePolicy{})

	got, err := repo.LoadBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.SessionID != memory.SessionID {
		t.Fatalf("expected session memory id %q, got %q", memory.SessionID, got.SessionID)
	}
	if store.getCalls != 0 {
		t.Fatalf("expected store not to be called, got %d", store.getCalls)
	}
}

func TestSessionMemoryRepositoryLoadFallsBackToStoreAndBackfillsCache(t *testing.T) {
	memory := newTestSessionMemory()
	cache := &fakeSessionMemoryCache{getErr: repository.ErrCacheMiss}
	store := &fakeSessionMemoryStore{memory: memory}
	repo := NewCompositeSessionMemoryRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	got, err := repo.LoadBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.SessionID != memory.SessionID {
		t.Fatalf("expected session memory id %q, got %q", memory.SessionID, got.SessionID)
	}
	if store.getCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.getCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache set to be called once, got %d", cache.setCalls)
	}
}

func TestSessionMemoryRepositoryLoadReturnsNotFoundForNotFoundMarker(t *testing.T) {
	cache := &fakeSessionMemoryCache{getErr: repository.ErrCacheNotFoundMarker}
	store := &fakeSessionMemoryStore{}
	repo := NewCompositeSessionMemoryRepository(store, cache, CachePolicy{})

	_, err := repo.LoadBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrSessionMemoryNotFound) {
		t.Fatalf("expected ErrSessionMemoryNotFound, got %v", err)
	}
}

func TestSessionMemoryRepositorySaveWritesStoreAndDeletesCache(t *testing.T) {
	memory := newTestSessionMemory()
	cache := &fakeSessionMemoryCache{}
	store := &fakeSessionMemoryStore{}
	repo := NewCompositeSessionMemoryRepository(store, cache, CachePolicy{})

	if err := repo.Save(context.Background(), memory); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if store.saveCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.saveCalls)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected cache delete to be called once, got %d", cache.deleteCalls)
	}
}

func newTestSessionMemory() *model.SessionMemory {
	return &model.SessionMemory{
		SessionID:        "session-1",
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "the city 广场",
		CurrentObjective: "接取下水道任务",
		RecentKeyEvents:  []string{"创建角色"},
		UpdatedAt:        time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC),
	}
}

type fakeSessionMemoryStore struct {
	memory    *model.SessionMemory
	getErr    error
	saveErr   error
	getCalls  int
	saveCalls int
}

func (f *fakeSessionMemoryStore) SaveSessionMemory(ctx context.Context, memory *model.SessionMemory) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.memory = memory
	return nil
}

func (f *fakeSessionMemoryStore) GetSessionMemoryBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.memory, nil
}

type fakeSessionMemoryCache struct {
	memory            *model.SessionMemory
	getErr            error
	setErr            error
	setNotFoundErr    error
	deleteErr         error
	setCalls          int
	setNotFoundCalls  int
	deleteCalls       int
}

func (f *fakeSessionMemoryCache) GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.memory, nil
}

func (f *fakeSessionMemoryCache) Set(ctx context.Context, memory *model.SessionMemory, ttl time.Duration) error {
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.memory = memory
	return nil
}

func (f *fakeSessionMemoryCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	f.setNotFoundCalls++
	return f.setNotFoundErr
}

func (f *fakeSessionMemoryCache) DeleteBySessionID(ctx context.Context, sessionID string) error {
	f.deleteCalls++
	return f.deleteErr
}
