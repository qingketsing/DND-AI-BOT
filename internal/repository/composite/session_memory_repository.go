package composite

import (
	"context"
	"errors"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
)

// CompositeSessionMemoryRepository 将 PG 真相源和 Redis 缓存组合成统一的会话长期记忆仓库。
type CompositeSessionMemoryRepository struct {
	store  postgresstore.SessionMemoryStore
	cache  rediscache.SessionMemoryCache
	policy CachePolicy
	group  singleflightGroup
}

// NewCompositeSessionMemoryRepository 创建带缓存策略的会话长期记忆组合仓库。
func NewCompositeSessionMemoryRepository(
	store postgresstore.SessionMemoryStore,
	cache rediscache.SessionMemoryCache,
	policy CachePolicy,
) *CompositeSessionMemoryRepository {
	return &CompositeSessionMemoryRepository{
		store:  store,
		cache:  cache,
		policy: policy,
		group:  newSingleflightGroup(),
	}
}

// Save 先写 PostgreSQL，再删除 Redis 缓存，保证后续读取回源最新数据。
func (r *CompositeSessionMemoryRepository) Save(ctx context.Context, memory *model.SessionMemory) error {
	if err := r.store.SaveSessionMemory(ctx, memory); err != nil {
		return err
	}
	if r.cache != nil {
		_ = r.cache.DeleteBySessionID(ctx, memory.SessionID)
	}
	return nil
}

// LoadBySessionID 优先读取 Redis，miss 后单飞回源 PostgreSQL，并在成功后回填缓存。
func (r *CompositeSessionMemoryRepository) LoadBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	if r.cache != nil {
		memory, err := r.cache.GetBySessionID(ctx, sessionID)
		switch {
		case err == nil:
			return memory, nil
		case errors.Is(err, repository.ErrCacheNotFoundMarker):
			return nil, repository.ErrSessionMemoryNotFound
		case !errors.Is(err, repository.ErrCacheMiss):
			// 缓存异常时降级走 PG，不直接失败。
		}
	}

	value, err := r.group.Do(sessionID, func() (any, error) {
		memory, err := r.store.GetSessionMemoryBySessionID(ctx, sessionID)
		if err != nil {
			if errors.Is(err, repository.ErrSessionMemoryNotFound) && r.cache != nil {
				_ = r.cache.SetNotFound(ctx, sessionID, r.policy.NextNotFoundTTL())
			}
			return nil, err
		}
		if r.cache != nil {
			_ = r.cache.Set(ctx, memory, r.policy.NextTTL())
		}
		return memory, nil
	})
	if err != nil {
		return nil, err
	}

	return value.(*model.SessionMemory), nil
}
