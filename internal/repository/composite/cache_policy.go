package composite

import (
	"math/rand"
	"time"
)

// CachePolicy 统一描述缓存 TTL、空值缓存 TTL 和抖动策略。
type CachePolicy struct {
	BaseTTL     time.Duration
	NotFoundTTL time.Duration
	TTLJitter   time.Duration
}

// NextTTL 返回带随机抖动的正常缓存过期时间，用于缓解缓存雪崩。
func (p CachePolicy) NextTTL() time.Duration {
	if p.BaseTTL <= 0 {
		return 0
	}
	if p.TTLJitter <= 0 {
		return p.BaseTTL
	}

	return p.BaseTTL + time.Duration(rand.Int63n(int64(p.TTLJitter)))
}

// NextNotFoundTTL 返回空值缓存使用的过期时间。
func (p CachePolicy) NextNotFoundTTL() time.Duration {
	return p.NotFoundTTL
}
