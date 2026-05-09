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

func (r *MessageJobRepository) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobProcessing
		job.WorkerID = workerID
		job.StartedAt = timePtr(startedAt)
		job.UpdatedAt = startedAt
	})
}

func (r *MessageJobRepository) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobCompleted
		job.FinishedAt = timePtr(finishedAt)
		job.LatencyMS = latencyMS
		job.UpdatedAt = finishedAt
	})
}

func (r *MessageJobRepository) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobRetryableFailed
		job.FinishedAt = timePtr(finishedAt)
		job.LastErrorCode = errorCode
		job.LastErrorMessage = errorMessage
		job.UpdatedAt = finishedAt
	})
}

func (r *MessageJobRepository) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.Status = model.MessageJobFailed
		job.FinishedAt = timePtr(finishedAt)
		job.LastErrorCode = errorCode
		job.LastErrorMessage = errorMessage
		job.UpdatedAt = finishedAt
	})
}

func (r *MessageJobRepository) IncrementAttempt(ctx context.Context, jobID string) error {
	return r.update(ctx, jobID, func(job *model.MessageJob) {
		job.AttemptCount++
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
	return cloned
}

func timePtr(value time.Time) *time.Time {
	copied := value
	return &copied
}
