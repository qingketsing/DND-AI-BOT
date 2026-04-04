package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

const notFoundMarker = "__not_found__"

// RedisSessionCache 负责将会话快照缓存到 Redis。
type RedisSessionCache struct {
	client *goredis.Client
}

// NewRedisSessionCache 创建会话 Redis 缓存实现。
func NewRedisSessionCache(client *goredis.Client) *RedisSessionCache {
	return &RedisSessionCache{client: client}
}

// Get 从 Redis 读取会话快照，并区分 miss 与空值缓存。
func (c *RedisSessionCache) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	value, err := c.client.Get(ctx, sessionCacheKey(sessionID)).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, repository.ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	if value == notFoundMarker {
		return nil, repository.ErrCacheNotFoundMarker
	}

	var session model.Session
	if err := json.Unmarshal([]byte(value), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Set 将完整会话对象序列化后写入 Redis。
func (c *RedisSessionCache) Set(ctx context.Context, session *model.Session, ttl time.Duration) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, sessionCacheKey(session.ID), payload, ttl).Err()
}

// SetNotFound 将“不存在”标记写入 Redis，用于防止缓存穿透。
func (c *RedisSessionCache) SetNotFound(ctx context.Context, sessionID string, ttl time.Duration) error {
	return c.client.Set(ctx, sessionCacheKey(sessionID), notFoundMarker, ttl).Err()
}

// Delete 删除指定会话的缓存键。
func (c *RedisSessionCache) Delete(ctx context.Context, sessionID string) error {
	return c.client.Del(ctx, sessionCacheKey(sessionID)).Err()
}

// sessionCacheKey 构造统一的 Session 缓存 key。
func sessionCacheKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}
