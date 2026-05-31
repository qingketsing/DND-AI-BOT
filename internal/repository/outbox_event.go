package repository

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
)

type OutboxJobRepairCandidate struct {
	Event model.OutboxEvent
	Job   model.MessageJob
}

type OutboxEventRepository interface {
	Create(ctx context.Context, event model.OutboxEvent) error
	GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string, publishedAt time.Time) error
	MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, nextRetryAt time.Time, lastError string) error
	ListPublishedWithQueuedJobs(ctx context.Context, limit int) ([]OutboxJobRepairCandidate, error)
}
