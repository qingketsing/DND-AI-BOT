package redis

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

func TestRedisSessionMemoryCacheSetGetAndDelete(t *testing.T) {
	server := newFakeRedisServer(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisSessionMemoryCache(client)
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	memory := &model.SessionMemory{
		SessionID:        "session-1",
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "the city 广场",
		CurrentObjective: "寻找格伦",
		RecentKeyEvents:  []string{"创建角色"},
		UpdatedAt:        now,
	}

	if err := cache.Set(context.Background(), memory, time.Minute); err != nil {
		t.Fatalf("expected set to succeed, got %v", err)
	}
	got, err := cache.GetBySessionID(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}
	if got.CharacterSummary != memory.CharacterSummary || got.SceneSummary != memory.SceneSummary {
		t.Fatalf("expected memory to round-trip, got %+v", got)
	}

	if err := cache.DeleteBySessionID(context.Background(), "session-1"); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if _, err := cache.GetBySessionID(context.Background(), "session-1"); !errors.Is(err, repository.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestRedisSessionMemoryCacheDistinguishesMarkerAndMiss(t *testing.T) {
	server := newFakeRedisServer(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Set(context.Background(), sessionMemoryCacheKey("missing"), notFoundMarker, time.Minute).Err(); err != nil {
		t.Fatalf("expected set marker to succeed, got %v", err)
	}

	cache := NewRedisSessionMemoryCache(client)
	_, err := cache.GetBySessionID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrCacheNotFoundMarker) {
		t.Fatalf("expected ErrCacheNotFoundMarker, got %v", err)
	}
}

func TestRedisSessionMemoryCacheRoundTripsJSON(t *testing.T) {
	server := newFakeRedisServer(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisSessionMemoryCache(client)
	memory := &model.SessionMemory{
		SessionID:        "session-1",
		CharacterSummary: "青稞，精灵法师。",
		SceneSummary:     "the city 广场",
		CurrentObjective: "寻找格伦",
		RecentKeyEvents:  []string{"创建角色"},
		UpdatedAt:        time.Unix(100, 0).UTC(),
	}
	if err := cache.Set(context.Background(), memory, time.Minute); err != nil {
		t.Fatalf("expected set to succeed, got %v", err)
	}

	raw := server.data[sessionMemoryCacheKey("session-1")]
	var got model.SessionMemory
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("expected cached payload to be valid json, got %v", err)
	}
	if got.CharacterSummary != memory.CharacterSummary || got.SceneSummary != memory.SceneSummary {
		t.Fatalf("expected json payload to round-trip, got %+v", got)
	}
}
