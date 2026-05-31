package repository

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
)

// MessageJobRepository 定义异步消息任务的持久化契约。
// 新建 queued 任务时应优先通过 AsyncMessageEnqueueRepository 在同一事务内与会话和 outbox 一起写入。
type MessageJobRepository interface {
	Create(ctx context.Context, job model.MessageJob) error
	GetByID(ctx context.Context, jobID string) (*model.MessageJob, error)
	GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error)
	MarkPublished(ctx context.Context, jobID string, publishedAt time.Time) error
	MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error
	MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error
	MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error
	MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error
	IncrementAttempt(ctx context.Context, jobID string) error
	ListStaleProcessing(ctx context.Context, cutoff time.Time, limit int) ([]model.MessageJob, error)
	ListRetryDue(ctx context.Context, now time.Time, limit int) ([]model.MessageJob, error)
	MarkRetryScheduled(ctx context.Context, jobID string, failedAt time.Time, nextRetryAt time.Time, errorCode string, errorMessage string) error
	RequeueRetryableWithOutbox(ctx context.Context, job model.MessageJob, event model.OutboxEvent, requeuedAt time.Time) error
	MarkHeartbeat(ctx context.Context, jobID string, heartbeatAt time.Time) error
	RepairPublished(ctx context.Context, jobID string, repairedAt time.Time) error
}
