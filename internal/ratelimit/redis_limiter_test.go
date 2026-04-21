package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisLimiterAllowsUntilPolicyLimitThenBlocks(t *testing.T) {
	client := &fakeRedisEvalClient{
		results: [][]interface{}{
			{int64(1), int64(60000)},
			{int64(2), int64(45000)},
			{int64(3), int64(30000)},
		},
	}
	limiter := NewRedisLimiter(client, "dnd:ratelimit")
	policy := Policy{Name: "message_user", Limit: 2, Window: time.Minute}
	now := testNow()

	first, err := limiter.Allow(context.Background(), "message:user:user-1", policy, now)
	if err != nil {
		t.Fatalf("expected first request to succeed, got %v", err)
	}
	if !first.Allowed || first.Remaining != 1 {
		t.Fatalf("expected first request allowed with 1 remaining, got %+v", first)
	}

	second, err := limiter.Allow(context.Background(), "message:user:user-1", policy, now)
	if err != nil {
		t.Fatalf("expected second request to succeed, got %v", err)
	}
	if !second.Allowed || second.Remaining != 0 {
		t.Fatalf("expected second request allowed with 0 remaining, got %+v", second)
	}

	third, err := limiter.Allow(context.Background(), "message:user:user-1", policy, now)
	if err != nil {
		t.Fatalf("expected third request to return decision, got %v", err)
	}
	if third.Allowed {
		t.Fatalf("expected third request to be blocked, got %+v", third)
	}
	if third.RetryAfter != 30*time.Second {
		t.Fatalf("expected retry after 30s, got %v", third.RetryAfter)
	}
	if got := client.keys[0][0]; got != "dnd:ratelimit:message_user:message:user:user-1" {
		t.Fatalf("expected redis key to include prefix and policy name, got %q", got)
	}
}

type fakeRedisEvalClient struct {
	results [][]interface{}
	keys    [][]string
	args    [][]interface{}
}

func (c *fakeRedisEvalClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	c.keys = append(c.keys, append([]string(nil), keys...))
	c.args = append(c.args, append([]interface{}(nil), args...))
	if len(c.results) == 0 {
		return redis.NewCmdResult([]interface{}{int64(1), int64(60000)}, nil)
	}
	result := c.results[0]
	c.results = c.results[1:]
	return redis.NewCmdResult(result, nil)
}
