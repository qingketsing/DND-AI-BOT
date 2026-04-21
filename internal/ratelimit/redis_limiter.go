package ratelimit

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const fixedWindowScript = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {current, ttl}
`

type redisEvalClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

type RedisLimiter struct {
	client redisEvalClient
	prefix string
}

func NewRedisLimiter(client redisEvalClient, prefix string) *RedisLimiter {
	prefix = strings.Trim(strings.TrimSpace(prefix), ":")
	if prefix == "" {
		prefix = "ratelimit"
	}
	return &RedisLimiter{client: client, prefix: prefix}
}

func (l *RedisLimiter) Allow(ctx context.Context, key string, policy Policy, now time.Time) (Decision, error) {
	key = strings.TrimSpace(key)
	if key == "" || policy.Limit <= 0 || policy.Window <= 0 {
		return Decision{Allowed: true, Key: key, PolicyName: policy.Name, Limit: policy.Limit}, nil
	}
	if l == nil || l.client == nil {
		return Decision{}, fmt.Errorf("rate limiter redis client is nil")
	}

	redisKey := buildRedisKey(l.prefix, key, policy, now)
	result, err := l.client.Eval(ctx, fixedWindowScript, []string{redisKey}, policy.Window.Milliseconds()).Result()
	if err != nil {
		return Decision{}, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		return Decision{}, fmt.Errorf("unexpected redis rate limit result: %v", result)
	}

	count, err := toInt64(values[0])
	if err != nil {
		return Decision{}, err
	}
	ttlMillis, err := toInt64(values[1])
	if err != nil {
		return Decision{}, err
	}
	retryAfter := time.Duration(maxInt64(ttlMillis, 0)) * time.Millisecond
	if retryAfter <= 0 {
		retryAfter = policy.Window
	}
	remaining := policy.Limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return Decision{
		Allowed:    int(count) <= policy.Limit,
		Key:        key,
		PolicyName: policy.Name,
		Limit:      policy.Limit,
		Remaining:  remaining,
		ResetAt:    now.Add(retryAfter),
		RetryAfter: retryAfter,
	}, nil
}

func buildRedisKey(prefix string, key string, policy Policy, now time.Time) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), ":")
	policyName := strings.TrimSpace(policy.Name)
	if policyName == "" {
		policyName = "default"
	}
	return prefix + ":" + policyName + ":" + strings.TrimSpace(key)
}

func toInt64(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint64:
		if typed > math.MaxInt64 {
			return 0, fmt.Errorf("integer overflows int64: %d", typed)
		}
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("unexpected integer type %T", value)
	}
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
