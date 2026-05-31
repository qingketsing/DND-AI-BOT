package memory

import (
	"context"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

// MessageJobRepository 在内存中保存异步消息任务状态。
type MessageJobRepository struct {
	asyncData *asyncMessageData
}

// NewMessageJobRepository 创建空的内存消息任务仓库。
func NewMessageJobRepository() *MessageJobRepository {
	return &MessageJobRepository{
		asyncData: newAsyncMessageData(),
	}
}

func (r *MessageJobRepository) Create(ctx context.Context, job model.MessageJob) error {
	_ = ctx
	if job.ID == "" || job.MessageID == "" {
		return ErrEmptySessionID
	}

	r.asyncData.mu.Lock()
	defer r.asyncData.mu.Unlock()

	r.asyncData.jobs[job.ID] = cloneMessageJob(job)
	r.asyncData.messageToJob[job.MessageID] = job.ID
	return nil
}

func (r *MessageJobRepository) GetByID(ctx context.Context, jobID string) (*model.MessageJob, error) {
	_ = ctx

	r.asyncData.mu.RLock()
	defer r.asyncData.mu.RUnlock()

	job, ok := r.asyncData.jobs[jobID]
	if !ok {
		return nil, repository.ErrMessageJobNotFound
	}
	cloned := cloneMessageJob(job)
	return &cloned, nil
}

func (r *MessageJobRepository) GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error) {
	_ = ctx

	r.asyncData.mu.RLock()
	jobID, ok := r.asyncData.messageToJob[messageID]
	if !ok {
		r.asyncData.mu.RUnlock()
		return nil, repository.ErrMessageJobNotFound
	}
	job := r.asyncData.jobs[jobID]
	r.asyncData.mu.RUnlock()

	cloned := cloneMessageJob(job)
	return &cloned, nil
}

func (r *MessageJobRepository) MarkPublished(ctx context.Context, jobID string, publishedAt time.Time) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobPublished
		job.NextRetryAt = nil
		job.UpdatedAt = publishedAt
	})
}

func (r *MessageJobRepository) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobProcessing
		job.WorkerID = workerID
		job.StartedAt = timePtr(startedAt)
		job.HeartbeatAt = timePtr(startedAt)
		job.NextRetryAt = nil
		job.UpdatedAt = startedAt
	})
}

func (r *MessageJobRepository) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobCompleted
		job.FinishedAt = timePtr(finishedAt)
		job.LatencyMS = latencyMS
		job.NextRetryAt = nil
		job.UpdatedAt = finishedAt
	})
}

func (r *MessageJobRepository) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobRetryableFailed
		job.FinishedAt = timePtr(finishedAt)
		job.LastErrorCode = errorCode
		job.LastErrorMessage = errorMessage
		job.NextRetryAt = nil
		job.UpdatedAt = finishedAt
	})
}

func (r *MessageJobRepository) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobFailed
		job.FinishedAt = timePtr(finishedAt)
		job.LastErrorCode = errorCode
		job.LastErrorMessage = errorMessage
		job.NextRetryAt = nil
		job.UpdatedAt = finishedAt
	})
}

func (r *MessageJobRepository) IncrementAttempt(ctx context.Context, jobID string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.AttemptCount++
	})
}

func (r *MessageJobRepository) ListStaleProcessing(ctx context.Context, cutoff time.Time, limit int) ([]model.MessageJob, error) {
	_ = ctx

	r.asyncData.mu.RLock()
	defer r.asyncData.mu.RUnlock()

	jobs := make([]model.MessageJob, 0)
	for _, job := range r.asyncData.jobs {
		if job.Status != model.MessageJobProcessing {
			continue
		}
		lastBeat := job.UpdatedAt
		if job.HeartbeatAt != nil {
			lastBeat = *job.HeartbeatAt
		}
		if !lastBeat.Before(cutoff) {
			continue
		}
		jobs = append(jobs, cloneMessageJob(job))
		if limit > 0 && len(jobs) >= limit {
			break
		}
	}
	return jobs, nil
}

func (r *MessageJobRepository) ListRetryDue(ctx context.Context, now time.Time, limit int) ([]model.MessageJob, error) {
	_ = ctx

	r.asyncData.mu.RLock()
	defer r.asyncData.mu.RUnlock()

	jobs := make([]model.MessageJob, 0)
	for _, job := range r.asyncData.jobs {
		if job.Status != model.MessageJobRetryableFailed || job.NextRetryAt == nil {
			continue
		}
		if job.NextRetryAt.After(now) || job.AttemptCount >= job.MaxAttempts {
			continue
		}
		jobs = append(jobs, cloneMessageJob(job))
		if limit > 0 && len(jobs) >= limit {
			break
		}
	}
	return jobs, nil
}

func (r *MessageJobRepository) MarkRetryScheduled(ctx context.Context, jobID string, failedAt time.Time, nextRetryAt time.Time, errorCode string, errorMessage string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobRetryableFailed
		job.FinishedAt = timePtr(failedAt)
		job.NextRetryAt = timePtr(nextRetryAt)
		job.LastErrorCode = errorCode
		job.LastErrorMessage = errorMessage
		job.UpdatedAt = failedAt
	})
}

func (r *MessageJobRepository) RequeueRetryableWithOutbox(ctx context.Context, job model.MessageJob, event model.OutboxEvent, requeuedAt time.Time) error {
	_ = event
	return r.update(ctx, job.ID, func(stored *model.MessageJob) {
		stored.Status = model.MessageJobQueued
		stored.WorkerID = ""
		stored.StartedAt = nil
		stored.FinishedAt = nil
		stored.NextRetryAt = nil
		stored.HeartbeatAt = nil
		stored.UpdatedAt = requeuedAt
	})
}

func (r *MessageJobRepository) MarkHeartbeat(ctx context.Context, jobID string, heartbeatAt time.Time) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.HeartbeatAt = timePtr(heartbeatAt)
		job.UpdatedAt = heartbeatAt
	})
}

func (r *MessageJobRepository) RepairPublished(ctx context.Context, jobID string, repairedAt time.Time) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobPublished
		job.UpdatedAt = repairedAt
	})
}

func (r *MessageJobRepository) update(ctx context.Context, jobID string, mutate func(*model.MessageJob)) error {
	_ = ctx

	r.asyncData.mu.Lock()
	defer r.asyncData.mu.Unlock()

	job, ok := r.asyncData.jobs[jobID]
	if !ok {
		return repository.ErrMessageJobNotFound
	}
	mutate(&job)
	r.asyncData.jobs[jobID] = cloneMessageJob(job)
	return nil
}

func cloneMessageJob(job model.MessageJob) model.MessageJob {
	cloned := job
	if job.StartedAt != nil {
		startedAt := *job.StartedAt
		cloned.StartedAt = &startedAt
	}
	if job.FinishedAt != nil {
		finishedAt := *job.FinishedAt
		cloned.FinishedAt = &finishedAt
	}
	if job.NextRetryAt != nil {
		nextRetryAt := *job.NextRetryAt
		cloned.NextRetryAt = &nextRetryAt
	}
	if job.HeartbeatAt != nil {
		heartbeatAt := *job.HeartbeatAt
		cloned.HeartbeatAt = &heartbeatAt
	}
	return cloned
}

func timePtr(value time.Time) *time.Time {
	copied := value
	return &copied
}
