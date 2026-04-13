package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

// RedisSessionMemoryCache 负责将会话长期记忆缓存到 Redis。
type RedisSessionMemoryCache struct {
	client *goredis.Client
}

// NewRedisSessionMemoryCache 创建会话长期记忆 Redis 缓存实现。
func NewRedisSessionMemoryCache(client *goredis.Client) *RedisSessionMemoryCache {
	return &RedisSessionMemoryCache{client: client}
}

// GetBySessionID 从 Redis 中读取会话长期记忆，并区分 miss 与空值缓存。
func (c *RedisSessionMemoryCache) GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	value, err := c.client.Get(ctx, sessionMemoryCacheKey(sessionID)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, repository.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	if value == notFoundMarker {
		return nil, repository.ErrCacheNotFoundMarker
	}

	var memory model.SessionMemory
	if err := json.Unmarshal([]byte(value), &memory); err != nil {
		return nil, err
	}

	return &memory, nil
}

// Set 将完整会话长期记忆对象序列化后写入 Redis。
func (c *RedisSessionMemoryCache) Set(ctx context.Context, memory *model.SessionMemory, ttl time.Duration) error {
	payload, err := json.Marshal(memory)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, sessionMemoryCacheKey(memory.SessionID), payload, ttl).Err()
}

// SetNotFound 将会话长期记忆不存在标记写入 Redis，用于防止缓存穿透。
func (c *RedisSessionMemoryCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	return c.client.Set(ctx, sessionMemoryCacheKey(sessionID), notFoundMarker, ttl).Err()
}

// DeleteBySessionID 删除指定会话对应的会话长期记忆缓存键。
func (c *RedisSessionMemoryCache) DeleteBySessionID(ctx context.Context, sessionID string) error {
	return c.client.Del(ctx, sessionMemoryCacheKey(sessionID)).Err()
}

func sessionMemoryCacheKey(sessionID string) string {
	return fmt.Sprintf("session_memory:%s", sessionID)
}
