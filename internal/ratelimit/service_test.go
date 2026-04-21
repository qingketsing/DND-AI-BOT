package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceCheckLoginBlocksWhenIPPolicyIsExceeded(t *testing.T) {
	limiter := &fakeLimiter{
		decisions: map[string]Decision{
			"login:ip:127.0.0.1": {
				Allowed:    false,
				Key:        "login:ip:127.0.0.1",
				PolicyName: "login_ip",
				Limit:      10,
				RetryAfter: 30 * time.Second,
			},
		},
	}
	service := NewService(limiter, testConfig(), fakeClock{now: testNow()})

	err := service.CheckLogin(context.Background(), CheckInput{
		IP:    "127.0.0.1",
		Email: "User@OpenCUMT.org",
	})

	if !IsRateLimited(err) {
		t.Fatalf("expected rate limited error, got %v", err)
	}
	if RetryAfter(err) != 30*time.Second {
		t.Fatalf("expected retry after 30s, got %v", RetryAfter(err))
	}
}

func TestServiceCheckLoginChecksAccountAfterIPAllowed(t *testing.T) {
	limiter := &fakeLimiter{
		decisions: map[string]Decision{
			"login:ip:127.0.0.1": {
				Allowed:    true,
				Key:        "login:ip:127.0.0.1",
				PolicyName: "login_ip",
				Limit:      10,
				Remaining:  9,
			},
			"login:account:user@opencumt.org": {
				Allowed:    false,
				Key:        "login:account:user@opencumt.org",
				PolicyName: "login_account",
				Limit:      5,
			},
		},
	}
	service := NewService(limiter, testConfig(), fakeClock{now: testNow()})

	err := service.CheckLogin(context.Background(), CheckInput{
		IP:    "127.0.0.1",
		Email: "User@OpenCUMT.org",
	})

	if !IsRateLimited(err) {
		t.Fatalf("expected account rate limited error, got %v", err)
	}
	if got := limiter.keys; len(got) != 2 || got[0] != "login:ip:127.0.0.1" || got[1] != "login:account:user@opencumt.org" {
		t.Fatalf("expected login checks to use ip then account keys, got %+v", got)
	}
}

func TestServiceCheckMessageUsesUserSessionAndIPPolicies(t *testing.T) {
	limiter := &fakeLimiter{
		defaultDecision: Decision{Allowed: true, Limit: 30},
	}
	service := NewService(limiter, testConfig(), fakeClock{now: testNow()})

	err := service.CheckMessage(context.Background(), CheckInput{
		IP:        "127.0.0.1",
		UserID:    "user-1",
		SessionID: "session-1",
	})

	if err != nil {
		t.Fatalf("expected message check to pass, got %v", err)
	}
	want := []string{
		"message:user:user-1",
		"message:session:session-1",
		"message:ip:127.0.0.1",
	}
	if len(limiter.keys) != len(want) {
		t.Fatalf("expected keys %+v, got %+v", want, limiter.keys)
	}
	for i := range want {
		if limiter.keys[i] != want[i] {
			t.Fatalf("expected key[%d]=%q, got %q", i, want[i], limiter.keys[i])
		}
	}
}

func TestServiceDisabledConfigAllowsWithoutCallingLimiter(t *testing.T) {
	limiter := &fakeLimiter{}
	config := testConfig()
	config.Enabled = false
	service := NewService(limiter, config, fakeClock{now: testNow()})

	if err := service.CheckRegister(context.Background(), CheckInput{IP: "127.0.0.1", Email: "user@opencumt.org"}); err != nil {
		t.Fatalf("expected disabled limiter to allow request, got %v", err)
	}
	if len(limiter.keys) != 0 {
		t.Fatalf("expected disabled limiter not to call backend, got keys %+v", limiter.keys)
	}
}

func TestServicePropagatesLimiterErrors(t *testing.T) {
	expected := errors.New("redis down")
	limiter := &fakeLimiter{err: expected}
	service := NewService(limiter, testConfig(), fakeClock{now: testNow()})

	err := service.CheckMessage(context.Background(), CheckInput{UserID: "user-1", SessionID: "session-1"})

	if !errors.Is(err, expected) {
		t.Fatalf("expected limiter error %v, got %v", expected, err)
	}
}

type fakeLimiter struct {
	decisions       map[string]Decision
	defaultDecision Decision
	err             error
	keys            []string
}

func (f *fakeLimiter) Allow(ctx context.Context, key string, policy Policy, now time.Time) (Decision, error) {
	f.keys = append(f.keys, key)
	if f.err != nil {
		return Decision{}, f.err
	}
	if decision, ok := f.decisions[key]; ok {
		return decision, nil
	}
	decision := f.defaultDecision
	decision.Key = key
	decision.PolicyName = policy.Name
	decision.Limit = policy.Limit
	return decision, nil
}

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time {
	return c.now
}

func testConfig() Config {
	return Config{
		Enabled: true,

		LoginIPLimit:       10,
		LoginIPWindow:      time.Minute,
		LoginAccountLimit:  5,
		LoginAccountWindow: 5 * time.Minute,

		RegisterIPLimit:     5,
		RegisterIPWindow:    time.Hour,
		RegisterEmailLimit:  3,
		RegisterEmailWindow: time.Hour,

		MessageUserLimit:     30,
		MessageUserWindow:    time.Minute,
		MessageSessionLimit:  20,
		MessageSessionWindow: time.Minute,
		MessageIPLimit:       60,
		MessageIPWindow:      time.Minute,
	}
}

func testNow() time.Time {
	return time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
}
