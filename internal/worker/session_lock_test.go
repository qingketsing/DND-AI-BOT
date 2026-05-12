package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSessionLockAcquireRenewRelease(t *testing.T) {
	client := newFakeLockClient()
	lock := newRedisSessionLockWithBackend(client)
	ctx := context.Background()

	acquired, err := lock.Acquire(ctx, "session-1", "job-1", "worker-1", time.Minute)
	if err != nil {
		t.Fatalf("expected acquire to succeed, got %v", err)
	}
	if !acquired {
		t.Fatal("expected acquire to return true")
	}

	acquired, err = lock.Acquire(ctx, "session-1", "job-2", "worker-2", time.Minute)
	if err != nil {
		t.Fatalf("expected second acquire to succeed, got %v", err)
	}
	if acquired {
		t.Fatal("expected second acquire to be rejected")
	}

	if err := lock.Renew(ctx, "session-1", "job-1", "worker-1", 2*time.Minute); err != nil {
		t.Fatalf("expected renew to succeed, got %v", err)
	}

	if err := lock.Release(ctx, "session-1", "job-1", "worker-1"); err != nil {
		t.Fatalf("expected release to succeed, got %v", err)
	}

	acquired, err = lock.Acquire(ctx, "session-1", "job-3", "worker-3", time.Minute)
	if err != nil {
		t.Fatalf("expected third acquire to succeed, got %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be available after release")
	}
}

func TestSessionLockReleaseIgnoresForeignOwner(t *testing.T) {
	client := newFakeLockClient()
	lock := newRedisSessionLockWithBackend(client)
	ctx := context.Background()

	acquired, err := lock.Acquire(ctx, "session-2", "job-10", "worker-a", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("expected initial acquire to succeed, got acquired=%v err=%v", acquired, err)
	}

	if err := lock.Release(ctx, "session-2", "job-10", "worker-b"); err != nil {
		t.Fatalf("expected foreign release to be ignored, got %v", err)
	}

	acquired, err = lock.Acquire(ctx, "session-2", "job-11", "worker-c", time.Minute)
	if err != nil {
		t.Fatalf("expected second acquire attempt to succeed, got %v", err)
	}
	if acquired {
		t.Fatal("expected foreign release to preserve the original lock")
	}
}

func TestSessionLockHeartbeatRenewsUntilStopped(t *testing.T) {
	client := newFakeLockClient()
	client.expireSignal = make(chan struct{}, 2)
	lock := newRedisSessionLockWithBackend(client)
	ctx := context.Background()

	acquired, err := lock.Acquire(ctx, "session-heartbeat", "job-1", "worker-1", 3*time.Minute)
	if err != nil || !acquired {
		t.Fatalf("expected initial acquire to succeed, got acquired=%v err=%v", acquired, err)
	}

	ticks := make(chan time.Time, 2)
	heartbeatCtx, cancel := context.WithCancel(ctx)
	errs := startSessionLockHeartbeat(
		heartbeatCtx,
		lock,
		"session-heartbeat",
		"job-1",
		"worker-1",
		3*time.Minute,
		time.Second,
		func(time.Duration) <-chan time.Time { return ticks },
	)

	ticks <- time.Now()
	ticks <- time.Now().Add(time.Second)
	<-client.expireSignal
	<-client.expireSignal
	cancel()

	for err := range errs {
		if err != nil {
			t.Fatalf("expected heartbeat to stop cleanly, got %v", err)
		}
	}

	if client.expireCalls != 2 {
		t.Fatalf("expected 2 renewals, got %d", client.expireCalls)
	}
}

func TestSessionLockHeartbeatReportsRenewFailure(t *testing.T) {
	client := newFakeLockClient()
	lock := newRedisSessionLockWithBackend(client)
	ctx := context.Background()

	acquired, err := lock.Acquire(ctx, "session-heartbeat-fail", "job-1", "worker-1", 3*time.Minute)
	if err != nil || !acquired {
		t.Fatalf("expected initial acquire to succeed, got acquired=%v err=%v", acquired, err)
	}

	client.compareAndExpireErr = errors.New("renew failed")
	ticks := make(chan time.Time, 1)
	errs := startSessionLockHeartbeat(
		ctx,
		lock,
		"session-heartbeat-fail",
		"job-1",
		"worker-1",
		3*time.Minute,
		time.Second,
		func(time.Duration) <-chan time.Time { return ticks },
	)

	ticks <- time.Now()

	err = <-errs
	if err == nil || err.Error() != "renew failed" {
		t.Fatalf("expected renew failure, got %v", err)
	}
}
