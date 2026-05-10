package service

import (
	"context"
	"time"

	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository"
)

const defaultOutboxDispatchBatchSize = 50

type OutboxDispatcher struct {
	outbox    repository.OutboxEventRepository
	jobs      repository.MessageJobRepository
	publisher queue.MessageJobPublisher
	batchSize int
}

func NewOutboxDispatcher(
	outbox repository.OutboxEventRepository,
	jobs repository.MessageJobRepository,
	publisher queue.MessageJobPublisher,
	batchSize int,
) *OutboxDispatcher {
	if batchSize <= 0 {
		batchSize = defaultOutboxDispatchBatchSize
	}
	return &OutboxDispatcher{
		outbox:    outbox,
		jobs:      jobs,
		publisher: publisher,
		batchSize: batchSize,
	}
}

func (d *OutboxDispatcher) DispatchOnce(ctx context.Context, now time.Time) (int, error) {
	events, err := d.outbox.GetPending(ctx, d.batchSize)
	if err != nil {
		return 0, err
	}

	dispatched := 0
	for _, event := range events {
		payload, err := queue.DecodeMessageJobPayload(event.PayloadJSON)
		if err != nil {
			if markErr := d.outbox.MarkFailedAttempt(ctx, event.ID, now, err.Error()); markErr != nil {
				return dispatched, markErr
			}
			continue
		}
		if err := d.publisher.Publish(ctx, payload); err != nil {
			if markErr := d.outbox.MarkFailedAttempt(ctx, event.ID, now, err.Error()); markErr != nil {
				return dispatched, markErr
			}
			continue
		}
		if err := d.outbox.MarkPublished(ctx, event.ID, now); err != nil {
			return dispatched, err
		}
		if err := d.jobs.MarkPublished(ctx, payload.JobID, now); err != nil {
			return dispatched, err
		}
		dispatched++
	}

	return dispatched, nil
}
