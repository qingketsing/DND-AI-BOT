package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

// RedisGameStateCache 负责将游戏进度快照缓存到 Redis。
type RedisGameStateCache struct {
	client *goredis.Client
}

// NewRedisGameStateCache 创建游戏进度 Redis 缓存实现。
func NewRedisGameStateCache(client *goredis.Client) *RedisGameStateCache {
	return &RedisGameStateCache{client: client}
}

// GetBySessionID 从 Redis 中读取游戏进度快照，并区分 miss 与空值缓存。
func (c *RedisGameStateCache) GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	value, err := c.client.Get(ctx, gameStateCacheKey(sessionID)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, repository.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	if value == notFoundMarker {
		return nil, repository.ErrCacheNotFoundMarker
	}

	var gameState state.GameState
	if err := json.Unmarshal([]byte(value), &gameState); err != nil {
		return nil, err
	}

	return &gameState, nil
}

// Set 将完整游戏进度对象序列化后写入 Redis。
func (c *RedisGameStateCache) Set(ctx context.Context, gameState *state.GameState, ttl time.Duration) error {
	payload, err := json.Marshal(gameState)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, gameStateCacheKey(gameState.SessionID), payload, ttl).Err()
}

// SetNotFound 将游戏进度不存在标记写入 Redis，用于防止缓存穿透。
func (c *RedisGameStateCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	return c.client.Set(ctx, gameStateCacheKey(sessionID), notFoundMarker, ttl).Err()
}

// DeleteBySessionID 删除指定会话对应的游戏进度缓存键。
func (c *RedisGameStateCache) DeleteBySessionID(ctx context.Context, sessionID string) error {
	return c.client.Del(ctx, gameStateCacheKey(sessionID)).Err()
}

// gameStateCacheKey 构造统一的游戏进度缓存 key。
func gameStateCacheKey(sessionID string) string {
	return fmt.Sprintf("game_state:%s", sessionID)
}
