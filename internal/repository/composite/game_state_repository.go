package composite

import (
	"context"
	"errors"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
)

// CompositeGameStateRepository 将 PG 真相源和 Redis 缓存组合成统一的游戏进度仓库。
type CompositeGameStateRepository struct {
	store  postgresstore.GameStateStore
	cache  rediscache.GameStateCache
	policy CachePolicy
	group  singleflightGroup
}

// NewCompositeGameStateRepository 创建带缓存策略的游戏进度组合仓库。
func NewCompositeGameStateRepository(
	store postgresstore.GameStateStore,
	cache rediscache.GameStateCache,
	policy CachePolicy,
) *CompositeGameStateRepository {
	return &CompositeGameStateRepository{
		store:  store,
		cache:  cache,
		policy: policy,
		group:  newSingleflightGroup(),
	}
}

// Save 先写 PostgreSQL，再删除 Redis 缓存，保证后续读取回源最新数据。
func (r *CompositeGameStateRepository) Save(ctx context.Context, state *state.GameState) error {
	if err := r.store.UpsertGameState(ctx, state); err != nil {
		return err
	}
	if r.cache != nil {
		_ = r.cache.DeleteBySessionID(ctx, state.SessionID)
	}
	return nil
}

// LoadBySessionID 优先读取 Redis，miss 后单飞回源 PostgreSQL，并在成功后回填缓存。
func (r *CompositeGameStateRepository) LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	if r.cache != nil {
		gameState, err := r.cache.GetBySessionID(ctx, sessionID)
		switch {
		case err == nil:
			return gameState, nil
		case errors.Is(err, repository.ErrCacheNotFoundMarker):
			return nil, repository.ErrGameStateNotFound
		case !errors.Is(err, repository.ErrCacheMiss):
			// 缓存异常时降级走 PG，不直接失败。
		}
	}

	value, err := r.group.Do(sessionID, func() (any, error) {
		gameState, err := r.store.GetGameStateBySessionID(ctx, sessionID)
		if err != nil {
			if errors.Is(err, repository.ErrGameStateNotFound) && r.cache != nil {
				_ = r.cache.SetNotFound(ctx, sessionID, r.policy.NextNotFoundTTL())
			}
			return nil, err
		}
		if r.cache != nil {
			_ = r.cache.Set(ctx, gameState, r.policy.NextTTL())
		}
		return gameState, nil
	})
	if err != nil {
		return nil, err
	}

	return value.(*state.GameState), nil
}

// DeleteBySessionID 删除指定会话的游戏进度，并清理 Redis 缓存。
func (r *CompositeGameStateRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	if err := r.store.DeleteGameStateBySessionID(ctx, sessionID); err != nil {
		return err
	}
	if r.cache != nil {
		_ = r.cache.DeleteBySessionID(ctx, sessionID)
	}
	return nil
}
