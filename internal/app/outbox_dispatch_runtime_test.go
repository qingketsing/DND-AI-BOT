package app

import (
	"context"
	"testing"
	"time"
)

func TestOutboxDispatchLoopStartInvokesDispatchOnce(t *testing.T) {
	dispatcher := &fakeOutboxDispatchOnceRunner{}
	loop := NewOutboxDispatchLoop(dispatcher, 10*time.Millisecond)

	if err := loop.Start(context.Background()); err != nil {
		t.Fatalf("expected start to succeed, got %v", err)
	}
	defer loop.Stop()

	deadline := time.After(time.Second)
	for {
		if dispatcher.calls > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("expected dispatch loop to invoke dispatcher")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

type fakeOutboxDispatchOnceRunner struct {
	calls int
}

func (f *fakeOutboxDispatchOnceRunner) DispatchOnce(ctx context.Context, now time.Time) (int, error) {
	_ = ctx
	_ = now
	f.calls++
	return 1, nil
}
