package ratelimit

import (
	"context"
	"errors"
	"time"
)

var ErrRateLimited = errors.New("rate limited")

type Policy struct {
	Name   string
	Limit  int
	Window time.Duration
}

type Decision struct {
	Allowed    bool
	Key        string
	PolicyName string
	Limit      int
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(ctx context.Context, key string, policy Policy, now time.Time) (Decision, error)
}

type RateLimitError struct {
	Decision Decision
}

func (e *RateLimitError) Error() string {
	return ErrRateLimited.Error()
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

func RetryAfter(err error) time.Duration {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Decision.RetryAfter
	}
	return 0
}
