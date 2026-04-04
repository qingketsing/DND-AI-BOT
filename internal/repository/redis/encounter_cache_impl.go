package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
	goredis "github.com/redis/go-redis/v9"
)

// RedisEncounterCache 负责将战斗快照缓存到 Redis。
type RedisEncounterCache struct {
	client *goredis.Client
}

// NewRedisEncounterCache 创建战斗 Redis 缓存实现。
func NewRedisEncounterCache(client *goredis.Client) *RedisEncounterCache {
	return &RedisEncounterCache{client: client}
}

// GetBySessionID 从 Redis 中读取战斗快照，并区分 miss 与空值缓存。
func (c *RedisEncounterCache) GetBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	value, err := c.client.Get(ctx, encounterCacheKey(sessionID)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, repository.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	if value == notFoundMarker {
		return nil, repository.ErrCacheNotFoundMarker
	}

	var encounter combat.Encounter
	if err := json.Unmarshal([]byte(value), &encounter); err != nil {
		return nil, err
	}

	return &encounter, nil
}

// Set 将完整战斗对象序列化后写入 Redis。
func (c *RedisEncounterCache) Set(ctx context.Context, encounter *combat.Encounter, ttl time.Duration) error {
	payload, err := json.Marshal(encounter)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, encounterCacheKey(encounter.SessionID), payload, ttl).Err()
}

// SetNotFound 将战斗不存在标记写入 Redis，用于防止缓存穿透。
func (c *RedisEncounterCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	return c.client.Set(ctx, encounterCacheKey(sessionID), notFoundMarker, ttl).Err()
}

// DeleteBySessionID 删除指定会话对应的战斗缓存键。
func (c *RedisEncounterCache) DeleteBySessionID(ctx context.Context, sessionID string) error {
	return c.client.Del(ctx, encounterCacheKey(sessionID)).Err()
}

// encounterCacheKey 构造统一的战斗缓存 key。
func encounterCacheKey(sessionID string) string {
	return fmt.Sprintf("encounter:%s", sessionID)
}
