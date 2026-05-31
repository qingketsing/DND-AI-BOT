package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"DND-AI-BOT/internal/service"
)

var ErrMissingAsyncRecovery = errors.New("missing async message recovery")

type asyncRecoveryOnceRunner interface {
	RunOnce(ctx context.Context) (service.AsyncRecoveryStats, error)
}

type AsyncRecoveryLoop struct {
	recovery asyncRecoveryOnceRunner
	interval time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewAsyncRecoveryLoop(recovery asyncRecoveryOnceRunner, interval time.Duration) *AsyncRecoveryLoop {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &AsyncRecoveryLoop{
		recovery: recovery,
		interval: interval,
	}
}

func (l *AsyncRecoveryLoop) Start(parent context.Context) error {
	if l == nil || l.recovery == nil {
		return ErrMissingAsyncRecovery
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancel != nil {
		return nil
	}

	ctx, cancel := context.WithCancel(parent)
	l.cancel = cancel
	go l.run(ctx)
	return nil
}

func (l *AsyncRecoveryLoop) Stop() {
	if l == nil {
		return
	}
	l.mu.Lock()
	cancel := l.cancel
	l.cancel = nil
	l.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (l *AsyncRecoveryLoop) run(ctx context.Context) {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	_, _ = l.recovery.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = l.recovery.RunOnce(ctx)
		}
	}
}
