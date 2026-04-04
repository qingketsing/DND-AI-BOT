package composite

import (
	"context"
	"errors"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
)

// CompositeEncounterRepository 将 PG 真相源和 Redis 缓存组合成统一的战斗仓库。
type CompositeEncounterRepository struct {
	store  postgresstore.EncounterStore
	cache  rediscache.EncounterCache
	policy CachePolicy
	group  singleflightGroup
}

// NewCompositeEncounterRepository 创建带缓存策略的战斗组合仓库。
func NewCompositeEncounterRepository(
	store postgresstore.EncounterStore,
	cache rediscache.EncounterCache,
	policy CachePolicy,
) *CompositeEncounterRepository {
	return &CompositeEncounterRepository{
		store:  store,
		cache:  cache,
		policy: policy,
		group:  newSingleflightGroup(),
	}
}

// Save 先写 PostgreSQL，再删除 Redis 缓存，保证后续读取回源最新数据。
func (r *CompositeEncounterRepository) Save(ctx context.Context, encounter *combat.Encounter) error {
	if err := r.store.UpsertEncounter(ctx, encounter); err != nil {
		return err
	}
	if r.cache != nil {
		_ = r.cache.DeleteBySessionID(ctx, encounter.SessionID)
	}
	return nil
}

// LoadBySessionID 优先读取 Redis，miss 后单飞回源 PostgreSQL，并在成功后回填缓存。
func (r *CompositeEncounterRepository) LoadBySessionID(ctx context.Context, sessionID string) (*combat.Encounter, error) {
	if r.cache != nil {
		encounter, err := r.cache.GetBySessionID(ctx, sessionID)
		switch {
		case err == nil:
			return encounter, nil
		case errors.Is(err, repository.ErrCacheNotFoundMarker):
			return nil, repository.ErrEncounterNotFound
		case !errors.Is(err, repository.ErrCacheMiss):
			// 缓存异常时降级走 PG，不直接失败。
		}
	}

	value, err := r.group.Do(sessionID, func() (any, error) {
		encounter, err := r.store.GetEncounterBySessionID(ctx, sessionID)
		if err != nil {
			if errors.Is(err, repository.ErrEncounterNotFound) && r.cache != nil {
				_ = r.cache.SetNotFound(ctx, sessionID, r.policy.NextNotFoundTTL())
			}
			return nil, err
		}
		if r.cache != nil {
			_ = r.cache.Set(ctx, encounter, r.policy.NextTTL())
		}
		return encounter, nil
	})
	if err != nil {
		return nil, err
	}

	return value.(*combat.Encounter), nil
}
