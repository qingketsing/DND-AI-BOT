package composite

import (
	"context"
	"errors"
	"sync"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
)

// CompositeSessionRepository 将 PG 真相源和 Redis 缓存组合成统一的会话仓库。
type CompositeSessionRepository struct {
	store  postgresstore.SessionStore
	cache  rediscache.SessionCache
	policy CachePolicy
	group  singleflightGroup
}

// NewCompositeSessionRepository 创建带缓存策略的会话组合仓库。
func NewCompositeSessionRepository(
	store postgresstore.SessionStore,
	cache rediscache.SessionCache,
	policy CachePolicy,
) *CompositeSessionRepository {
	return &CompositeSessionRepository{
		store:  store,
		cache:  cache,
		policy: policy,
		group:  newSingleflightGroup(),
	}
}

// Save 先写 PostgreSQL，再删除 Redis 缓存，保证后续读取回源最新数据。
func (r *CompositeSessionRepository) Save(ctx context.Context, session *model.Session) error {
	if err := r.store.UpsertSession(ctx, session); err != nil {
		return err
	}
	if r.cache != nil {
		_ = r.cache.Delete(ctx, session.ID)
	}

	return nil
}

// Load 优先读取 Redis，miss 后单飞回源 PostgreSQL，并在成功后回填缓存。
func (r *CompositeSessionRepository) Load(ctx context.Context, sessionID string) (*model.Session, error) {
	if r.cache != nil {
		session, err := r.cache.Get(ctx, sessionID)
		switch {
		case err == nil:
			return session, nil
		case errors.Is(err, repository.ErrCacheNotFoundMarker):
			return nil, repository.ErrSessionNotFound
		case !errors.Is(err, repository.ErrCacheMiss):
			// 缓存异常时降级走 PG，不直接失败。
		}
	}

	value, err := r.group.Do(sessionID, func() (any, error) {
		session, err := r.store.GetSession(ctx, sessionID)
		if err != nil {
			if errors.Is(err, repository.ErrSessionNotFound) && r.cache != nil {
				_ = r.cache.SetNotFound(ctx, sessionID, r.policy.NextNotFoundTTL())
			}
			return nil, err
		}
		if r.cache != nil {
			_ = r.cache.Set(ctx, session, r.policy.NextTTL())
		}
		return session, nil
	})
	if err != nil {
		return nil, err
	}

	return value.(*model.Session), nil
}

// ListByUserID 直接从 PostgreSQL 真相源加载指定用户的会话列表。
func (r *CompositeSessionRepository) ListByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	return r.store.ListSessionsByUserID(ctx, userID)
}

// singleflightGroup 是最小实现的单飞结构，用于防止缓存击穿时重复回源。
type singleflightGroup struct {
	mu    sync.Mutex
	calls map[string]*singleflightCall
}

// singleflightCall 表示同一个 key 的一次在途回源请求。
type singleflightCall struct {
	wg  sync.WaitGroup
	val any
	err error
}

// newSingleflightGroup 创建空的单飞控制器。
func newSingleflightGroup() singleflightGroup {
	return singleflightGroup{
		calls: make(map[string]*singleflightCall),
	}
}

// Do 保证同一个 key 的回源逻辑在同一时刻只执行一次。
func (g *singleflightGroup) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		call.wg.Wait()
		return call.val, call.err
	}

	call := &singleflightCall{}
	call.wg.Add(1)
	g.calls[key] = call
	g.mu.Unlock()

	call.val, call.err = fn()
	call.wg.Done()

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return call.val, call.err
}
