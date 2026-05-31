package postgres

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

type OutboxEventStore interface {
	Create(ctx context.Context, event model.OutboxEvent) error
	GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string, publishedAt time.Time) error
	MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, nextRetryAt time.Time, lastError string) error
	ListPublishedWithQueuedJobs(ctx context.Context, limit int) ([]repository.OutboxJobRepairCandidate, error)
}
