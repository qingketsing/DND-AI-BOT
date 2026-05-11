package app

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrMissingOutboxDispatcher = errors.New("missing outbox dispatcher")

type outboxDispatchOnceRunner interface {
	DispatchOnce(ctx context.Context, now time.Time) (int, error)
}

type OutboxDispatchLoop struct {
	dispatcher outboxDispatchOnceRunner
	interval   time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewOutboxDispatchLoop(dispatcher outboxDispatchOnceRunner, interval time.Duration) *OutboxDispatchLoop {
	if interval <= 0 {
		interval = time.Second
	}
	return &OutboxDispatchLoop{
		dispatcher: dispatcher,
		interval:   interval,
	}
}

func (l *OutboxDispatchLoop) Start(parent context.Context) error {
	if l == nil || l.dispatcher == nil {
		return ErrMissingOutboxDispatcher
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

func (l *OutboxDispatchLoop) Stop() {
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

func (l *OutboxDispatchLoop) run(ctx context.Context) {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	_, _ = l.dispatcher.DispatchOnce(ctx, time.Now().UTC())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = l.dispatcher.DispatchOnce(ctx, time.Now().UTC())
		}
	}
}
