package service

import (
	"context"
	"fmt"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository"
)

const (
	defaultAsyncRecoveryBatchSize            = 50
	defaultAsyncRecoveryRetryDelay           = 30 * time.Second
	defaultAsyncRecoveryProcessingStaleAfter = 5 * time.Minute
)

type AsyncRecoveryConfig struct {
	BatchSize            int
	RetryDelay           time.Duration
	ProcessingStaleAfter time.Duration
}

type AsyncRecoveryStats struct {
	StaleScheduled     int
	StaleSkippedLocked int
	RetryRequeued      int
	PublishedRepaired  int
	PermanentFailed    int
}

type AsyncMessageRecovery struct {
	jobs   repository.MessageJobRepository
	outbox repository.OutboxEventRepository
	locks  repository.SessionLockInspector
	now    func() time.Time
	config AsyncRecoveryConfig
}

type AsyncMessageRecoveryOption func(*AsyncMessageRecovery)

func WithAsyncMessageRecoveryClock(now func() time.Time) AsyncMessageRecoveryOption {
	return func(r *AsyncMessageRecovery) {
		if now != nil {
			r.now = now
		}
	}
}

func NewAsyncMessageRecovery(
	jobs repository.MessageJobRepository,
	outbox repository.OutboxEventRepository,
	locks repository.SessionLockInspector,
	config AsyncRecoveryConfig,
	options ...AsyncMessageRecoveryOption,
) *AsyncMessageRecovery {
	config = normalizeAsyncRecoveryConfig(config)
	recovery := &AsyncMessageRecovery{
		jobs:   jobs,
		outbox: outbox,
		locks:  locks,
		now:    func() time.Time { return time.Now().UTC() },
		config: config,
	}
	for _, option := range options {
		option(recovery)
	}
	return recovery
}

func (r *AsyncMessageRecovery) RunOnce(ctx context.Context) (AsyncRecoveryStats, error) {
	if r.jobs == nil || r.outbox == nil || r.locks == nil {
		return AsyncRecoveryStats{}, fmt.Errorf("async message recovery dependencies are not configured")
	}

	now := r.now().UTC()
	stats := AsyncRecoveryStats{}
	if err := r.recoverStaleProcessing(ctx, now, &stats); err != nil {
		return stats, err
	}
	if err := r.requeueDueRetries(ctx, now, &stats); err != nil {
		return stats, err
	}
	if err := r.repairPublishedOutbox(ctx, now, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *AsyncMessageRecovery) recoverStaleProcessing(ctx context.Context, now time.Time, stats *AsyncRecoveryStats) error {
	cutoff := now.Add(-r.config.ProcessingStaleAfter)
	jobs, err := r.jobs.ListStaleProcessing(ctx, cutoff, r.config.BatchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		owner, err := r.locks.Inspect(ctx, job.SessionID)
		if err != nil {
			return err
		}
		if owner.Exists {
			stats.StaleSkippedLocked++
			continue
		}
		if job.AttemptCount >= job.MaxAttempts {
			if err := r.jobs.MarkFailed(ctx, job.ID, now, "max_attempts_exceeded", "processing job exceeded max attempts during recovery"); err != nil {
				return err
			}
			stats.PermanentFailed++
			continue
		}
		if err := r.jobs.MarkRetryScheduled(ctx, job.ID, now, now.Add(r.config.RetryDelay), "processing_stale", "processing heartbeat is stale"); err != nil {
			return err
		}
		stats.StaleScheduled++
	}
	return nil
}

func (r *AsyncMessageRecovery) requeueDueRetries(ctx context.Context, now time.Time, stats *AsyncRecoveryStats) error {
	jobs, err := r.jobs.ListRetryDue(ctx, now, r.config.BatchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		event, err := recoveryOutboxEvent(job, now)
		if err != nil {
			return err
		}
		if err := r.jobs.RequeueRetryableWithOutbox(ctx, job, event, now); err != nil {
			return err
		}
		stats.RetryRequeued++
	}
	return nil
}

func (r *AsyncMessageRecovery) repairPublishedOutbox(ctx context.Context, now time.Time, stats *AsyncRecoveryStats) error {
	candidates, err := r.outbox.ListPublishedWithQueuedJobs(ctx, r.config.BatchSize)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := r.jobs.RepairPublished(ctx, candidate.Job.ID, now); err != nil {
			return err
		}
		stats.PublishedRepaired++
	}
	return nil
}

func recoveryOutboxEvent(job model.MessageJob, now time.Time) (model.OutboxEvent, error) {
	payload, err := queue.EncodeMessageJobPayload(queue.MessageJobPayload{
		JobID:     job.ID,
		MessageID: job.MessageID,
		SessionID: job.SessionID,
		UserID:    job.UserID,
		Attempt:   job.AttemptCount + 1,
		QueuedAt:  now,
	})
	if err != nil {
		return model.OutboxEvent{}, err
	}
	return model.OutboxEvent{
		ID:            fmt.Sprintf("outbox-retry-%s-%d", job.ID, now.UnixNano()),
		AggregateType: "message_job",
		AggregateID:   job.ID,
		EventType:     "message_job_queued",
		PayloadJSON:   payload,
		Status:        model.OutboxEventPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func normalizeAsyncRecoveryConfig(config AsyncRecoveryConfig) AsyncRecoveryConfig {
	if config.BatchSize <= 0 {
		config.BatchSize = defaultAsyncRecoveryBatchSize
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = defaultAsyncRecoveryRetryDelay
	}
	if config.ProcessingStaleAfter <= 0 {
		config.ProcessingStaleAfter = defaultAsyncRecoveryProcessingStaleAfter
	}
	return config
}
