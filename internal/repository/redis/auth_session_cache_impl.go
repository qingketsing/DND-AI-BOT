package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"DND-AI-BOT/internal/repository"
	goredis "github.com/redis/go-redis/v9"
)

// RedisAuthSessionCache stores auth session projections in Redis.
type RedisAuthSessionCache struct {
	client goredis.UniversalClient
}

// NewRedisAuthSessionCache creates a Redis-backed auth session cache.
func NewRedisAuthSessionCache(client goredis.UniversalClient) *RedisAuthSessionCache {
	return &RedisAuthSessionCache{client: client}
}

// Get loads a cached auth session projection by token hash.
func (c *RedisAuthSessionCache) Get(ctx context.Context, tokenHash string) (CachedAuthSession, error) {
	value, err := c.client.Get(ctx, authSessionCacheKey(tokenHash)).Result()
	if errors.Is(err, goredis.Nil) {
		return CachedAuthSession{}, repository.ErrCacheMiss
	}
	if err != nil {
		return CachedAuthSession{}, err
	}
	if value == notFoundMarker {
		return CachedAuthSession{}, repository.ErrCacheNotFoundMarker
	}

	var session CachedAuthSession
	if err := json.Unmarshal([]byte(value), &session); err != nil {
		return CachedAuthSession{}, err
	}
	return session, nil
}

// Set stores a cached auth session projection by token hash.
func (c *RedisAuthSessionCache) Set(ctx context.Context, tokenHash string, session CachedAuthSession, ttl time.Duration) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, authSessionCacheKey(tokenHash), payload, ttl).Err()
}

// Delete removes a token-hash lookup from cache.
func (c *RedisAuthSessionCache) Delete(ctx context.Context, tokenHash string) error {
	return c.client.Del(ctx, authSessionCacheKey(tokenHash)).Err()
}

func authSessionCacheKey(tokenHash string) string {
	return fmt.Sprintf("auth:session:%s", tokenHash)
}
