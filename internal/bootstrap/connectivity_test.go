package bootstrap

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestLoadDependencyConfigReadsEnvironment(t *testing.T) {
	t.Setenv("POSTGRES_ADDR", "postgres:5432")
	t.Setenv("REDIS_ADDR", "redis:6379")

	config, err := LoadDependencyConfig()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}
	if config.PostgresAddr != "postgres:5432" {
		t.Fatalf("expected postgres address postgres:5432, got %q", config.PostgresAddr)
	}
	if config.RedisAddr != "redis:6379" {
		t.Fatalf("expected redis address redis:6379, got %q", config.RedisAddr)
	}
}

func TestLoadDependencyConfigRejectsMissingEnvironment(t *testing.T) {
	_ = os.Unsetenv("POSTGRES_ADDR")
	_ = os.Unsetenv("REDIS_ADDR")

	_, err := LoadDependencyConfig()
	if err == nil {
		t.Fatal("expected config load to fail when environment is missing")
	}
}

func TestCheckTCPConnectionSucceedsWhenListenerIsAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := checkTCPConnectionWithDialer(ctx, "postgres:5432", func(context.Context, string, string) (io.Closer, error) {
		return fakeCloser{}, nil
	}); err != nil {
		t.Fatalf("expected tcp check to succeed, got %v", err)
	}
}

func TestCheckTCPConnectionFailsForClosedAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := checkTCPConnectionWithDialer(ctx, "postgres:5432", func(context.Context, string, string) (io.Closer, error) {
		return nil, context.DeadlineExceeded
	})
	if err == nil {
		t.Fatal("expected tcp check to fail")
	}
}

type fakeCloser struct{}

func (fakeCloser) Close() error { return nil }
