package app

import (
	"context"
	"errors"
	"time"

	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/worker"
)

type inProcessAsyncMessagePublisher struct {
	processor  *worker.MessageJobProcessor
	jobs       chan queue.MessageJobPayload
	retryDelay time.Duration
}

func newInProcessAsyncMessagePublisher(
	processor *worker.MessageJobProcessor,
	workerCount int,
	queueBuffer int,
	retryDelay time.Duration,
) queue.MessageJobPublisher {
	publisher := &inProcessAsyncMessagePublisher{
		processor:  processor,
		jobs:       make(chan queue.MessageJobPayload, queueBuffer),
		retryDelay: retryDelay,
	}
	for i := 0; i < workerCount; i++ {
		go publisher.runWorker()
	}
	return publisher
}

func (p *inProcessAsyncMessagePublisher) Publish(ctx context.Context, payload queue.MessageJobPayload) error {
	if p.processor == nil {
		return errors.New("message job processor is nil")
	}
	select {
	case p.jobs <- payload:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *inProcessAsyncMessagePublisher) runWorker() {
	for payload := range p.jobs {
		if err := p.processor.ProcessMessageJob(context.Background(), payload); err != nil {
			if errors.Is(err, worker.ErrSessionBusy) {
				time.Sleep(p.retryDelay)
				p.jobs <- payload
			}
		}
	}
}
