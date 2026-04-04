package composite

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestSessionRepositoryLoadReturnsCachedValue(t *testing.T) {
	session := model.NewSession("session-1", model.ChannelWeb, time.Now().UTC())
	cache := &fakeSessionCache{session: session}
	store := &fakeSessionStore{}
	repo := NewCompositeSessionRepository(store, cache, CachePolicy{})

	got, err := repo.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.ID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", got.ID)
	}
	if store.getCalls != 0 {
		t.Fatalf("expected store not to be called, got %d", store.getCalls)
	}
}

func TestSessionRepositoryLoadFallsBackToStoreAndBackfillsCache(t *testing.T) {
	session := model.NewSession("session-1", model.ChannelWeb, time.Now().UTC())
	cache := &fakeSessionCache{getErr: repository.ErrCacheMiss}
	store := &fakeSessionStore{session: session}
	repo := NewCompositeSessionRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	got, err := repo.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if got.ID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", got.ID)
	}
	if store.getCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.getCalls)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache to be backfilled once, got %d", cache.setCalls)
	}
}

func TestSessionRepositoryLoadReturnsNotFoundForNotFoundMarker(t *testing.T) {
	cache := &fakeSessionCache{getErr: repository.ErrCacheNotFoundMarker}
	store := &fakeSessionStore{}
	repo := NewCompositeSessionRepository(store, cache, CachePolicy{})

	_, err := repo.Load(context.Background(), "missing")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionRepositoryLoadStoresNotFoundMarkerWhenStoreMisses(t *testing.T) {
	cache := &fakeSessionCache{getErr: repository.ErrCacheMiss}
	store := &fakeSessionStore{getErr: repository.ErrSessionNotFound}
	repo := NewCompositeSessionRepository(store, cache, CachePolicy{
		BaseTTL:     time.Minute,
		NotFoundTTL: 30 * time.Second,
	})

	_, err := repo.Load(context.Background(), "missing")
	if !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
	if cache.setNotFoundCalls != 1 {
		t.Fatalf("expected set not found to be called once, got %d", cache.setNotFoundCalls)
	}
}

func TestSessionRepositorySaveWritesStoreAndDeletesCache(t *testing.T) {
	session := model.NewSession("session-1", model.ChannelWeb, time.Now().UTC())
	cache := &fakeSessionCache{}
	store := &fakeSessionStore{}
	repo := NewCompositeSessionRepository(store, cache, CachePolicy{})

	if err := repo.Save(context.Background(), session); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if store.upsertCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.upsertCalls)
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected cache delete to be called once, got %d", cache.deleteCalls)
	}
}

type fakeSessionStore struct {
	session     *model.Session
	getErr      error
	upsertErr   error
	getCalls    int
	upsertCalls int
}

func (f *fakeSessionStore) UpsertSession(ctx context.Context, session *model.Session) error {
	f.upsertCalls++
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.session = session
	return nil
}

func (f *fakeSessionStore) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.session, nil
}

type fakeSessionCache struct {
	session          *model.Session
	getErr           error
	setErr           error
	setNotFoundErr   error
	deleteErr        error
	setCalls         int
	setNotFoundCalls int
	deleteCalls      int
}

func (f *fakeSessionCache) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.session, nil
}

func (f *fakeSessionCache) Set(ctx context.Context, session *model.Session, ttl time.Duration) error {
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	f.session = session
	return nil
}

func (f *fakeSessionCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	f.setNotFoundCalls++
	return f.setNotFoundErr
}

func (f *fakeSessionCache) Delete(ctx context.Context, sessionID string) error {
	f.deleteCalls++
	return f.deleteErr
}
