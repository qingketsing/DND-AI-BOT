package service

import (
	"context"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository"
)

func TestAsyncMessageRecoverySchedulesUnlockedStaleProcessing(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	jobs := &fakeRecoveryJobRepository{
		stale: []model.MessageJob{
			{
				ID:           "job-locked",
				SessionID:    "session-locked",
				Status:       model.MessageJobProcessing,
				AttemptCount: 1,
				MaxAttempts:  3,
			},
			{
				ID:           "job-stale",
				SessionID:    "session-stale",
				Status:       model.MessageJobProcessing,
				AttemptCount: 1,
				MaxAttempts:  3,
			},
			{
				ID:           "job-maxed",
				SessionID:    "session-maxed",
				Status:       model.MessageJobProcessing,
				AttemptCount: 3,
				MaxAttempts:  3,
			},
		},
	}
	recovery := NewAsyncMessageRecovery(
		jobs,
		&fakeRecoveryOutboxRepository{},
		&fakeRecoverySessionLock{owners: map[string]repository.SessionLockOwner{
			"session-locked": {Exists: true, JobID: "job-locked", WorkerID: "worker-1"},
		}},
		AsyncRecoveryConfig{RetryDelay: 30 * time.Second},
		WithAsyncMessageRecoveryClock(func() time.Time { return now }),
	)

	stats, err := recovery.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected recovery to succeed, got %v", err)
	}
	if stats.StaleSkippedLocked != 1 || stats.StaleScheduled != 1 || stats.PermanentFailed != 1 {
		t.Fatalf("unexpected recovery stats: %+v", stats)
	}
	if len(jobs.retryScheduled) != 1 || jobs.retryScheduled[0].jobID != "job-stale" {
		t.Fatalf("expected job-stale to be scheduled, got %+v", jobs.retryScheduled)
	}
	if !jobs.retryScheduled[0].nextRetryAt.Equal(now.Add(30 * time.Second)) {
		t.Fatalf("expected retry delay 30s, got %s", jobs.retryScheduled[0].nextRetryAt.Sub(now))
	}
	if len(jobs.failed) != 1 || jobs.failed[0] != "job-maxed" {
		t.Fatalf("expected maxed job to fail permanently, got %+v", jobs.failed)
	}
}

func TestAsyncMessageRecoveryRequeuesDueRetryWithOutbox(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 10, 0, 0, time.UTC)
	job := model.MessageJob{
		ID:           "job-retry",
		MessageID:    "msg-retry",
		SessionID:    "session-retry",
		UserID:       "user-retry",
		Status:       model.MessageJobRetryableFailed,
		AttemptCount: 1,
		MaxAttempts:  3,
	}
	jobs := &fakeRecoveryJobRepository{retryDue: []model.MessageJob{job}}
	recovery := NewAsyncMessageRecovery(
		jobs,
		&fakeRecoveryOutboxRepository{},
		&fakeRecoverySessionLock{},
		AsyncRecoveryConfig{},
		WithAsyncMessageRecoveryClock(func() time.Time { return now }),
	)

	stats, err := recovery.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected recovery to succeed, got %v", err)
	}
	if stats.RetryRequeued != 1 {
		t.Fatalf("expected one retry requeue, got %+v", stats)
	}
	if len(jobs.requeued) != 1 {
		t.Fatalf("expected one requeued job, got %+v", jobs.requeued)
	}
	event := jobs.requeued[0].event
	if event.AggregateID != job.ID || event.Status != model.OutboxEventPending {
		t.Fatalf("unexpected requeue outbox event: %+v", event)
	}
	payload, err := queue.DecodeMessageJobPayload(event.PayloadJSON)
	if err != nil {
		t.Fatalf("expected payload decode to succeed, got %v", err)
	}
	if payload.JobID != job.ID || payload.Attempt != 2 || !payload.QueuedAt.Equal(now) {
		t.Fatalf("unexpected retry payload: %+v", payload)
	}
}

func TestAsyncMessageRecoveryRepairsPublishedOutboxQueuedJobs(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 20, 0, 0, time.UTC)
	jobs := &fakeRecoveryJobRepository{}
	outbox := &fakeRecoveryOutboxRepository{
		repairCandidates: []repository.OutboxJobRepairCandidate{{
			Job: model.MessageJob{ID: "job-repair", Status: model.MessageJobQueued},
		}},
	}
	recovery := NewAsyncMessageRecovery(
		jobs,
		outbox,
		&fakeRecoverySessionLock{},
		AsyncRecoveryConfig{},
		WithAsyncMessageRecoveryClock(func() time.Time { return now }),
	)

	stats, err := recovery.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected recovery to succeed, got %v", err)
	}
	if stats.PublishedRepaired != 1 {
		t.Fatalf("expected one repair, got %+v", stats)
	}
	if len(jobs.repaired) != 1 || jobs.repaired[0] != "job-repair" {
		t.Fatalf("expected job-repair to be repaired, got %+v", jobs.repaired)
	}
}

type fakeRecoveryJobRepository struct {
	stale          []model.MessageJob
	retryDue       []model.MessageJob
	retryScheduled []fakeRetrySchedule
	requeued       []fakeRequeuedJob
	repaired       []string
	failed         []string
}

type fakeRetrySchedule struct {
	jobID       string
	nextRetryAt time.Time
}

type fakeRequeuedJob struct {
	job   model.MessageJob
	event model.OutboxEvent
}

func (f *fakeRecoveryJobRepository) Create(ctx context.Context, job model.MessageJob) error {
	panic("unexpected Create call")
}

func (f *fakeRecoveryJobRepository) GetByID(ctx context.Context, jobID string) (*model.MessageJob, error) {
	panic("unexpected GetByID call")
}

func (f *fakeRecoveryJobRepository) GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error) {
	panic("unexpected GetByMessageID call")
}

func (f *fakeRecoveryJobRepository) MarkPublished(ctx context.Context, jobID string, publishedAt time.Time) error {
	panic("unexpected MarkPublished call")
}

func (f *fakeRecoveryJobRepository) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	panic("unexpected MarkProcessing call")
}

func (f *fakeRecoveryJobRepository) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	panic("unexpected MarkCompleted call")
}

func (f *fakeRecoveryJobRepository) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	panic("unexpected MarkRetryableFailed call")
}

func (f *fakeRecoveryJobRepository) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	f.failed = append(f.failed, jobID)
	return nil
}

func (f *fakeRecoveryJobRepository) IncrementAttempt(ctx context.Context, jobID string) error {
	panic("unexpected IncrementAttempt call")
}

func (f *fakeRecoveryJobRepository) ListStaleProcessing(ctx context.Context, cutoff time.Time, limit int) ([]model.MessageJob, error) {
	return append([]model.MessageJob(nil), f.stale...), nil
}

func (f *fakeRecoveryJobRepository) ListRetryDue(ctx context.Context, now time.Time, limit int) ([]model.MessageJob, error) {
	return append([]model.MessageJob(nil), f.retryDue...), nil
}

func (f *fakeRecoveryJobRepository) MarkRetryScheduled(ctx context.Context, jobID string, failedAt time.Time, nextRetryAt time.Time, errorCode string, errorMessage string) error {
	f.retryScheduled = append(f.retryScheduled, fakeRetrySchedule{jobID: jobID, nextRetryAt: nextRetryAt})
	return nil
}

func (f *fakeRecoveryJobRepository) RequeueRetryableWithOutbox(ctx context.Context, job model.MessageJob, event model.OutboxEvent, requeuedAt time.Time) error {
	f.requeued = append(f.requeued, fakeRequeuedJob{job: job, event: event})
	return nil
}

func (f *fakeRecoveryJobRepository) MarkHeartbeat(ctx context.Context, jobID string, heartbeatAt time.Time) error {
	panic("unexpected MarkHeartbeat call")
}

func (f *fakeRecoveryJobRepository) RepairPublished(ctx context.Context, jobID string, repairedAt time.Time) error {
	f.repaired = append(f.repaired, jobID)
	return nil
}

type fakeRecoveryOutboxRepository struct {
	repairCandidates []repository.OutboxJobRepairCandidate
}

func (f *fakeRecoveryOutboxRepository) Create(ctx context.Context, event model.OutboxEvent) error {
	panic("unexpected Create call")
}

func (f *fakeRecoveryOutboxRepository) GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	panic("unexpected GetPending call")
}

func (f *fakeRecoveryOutboxRepository) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	panic("unexpected MarkPublished call")
}

func (f *fakeRecoveryOutboxRepository) MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, nextRetryAt time.Time, lastError string) error {
	panic("unexpected MarkFailedAttempt call")
}

func (f *fakeRecoveryOutboxRepository) ListPublishedWithQueuedJobs(ctx context.Context, limit int) ([]repository.OutboxJobRepairCandidate, error) {
	return append([]repository.OutboxJobRepairCandidate(nil), f.repairCandidates...), nil
}

type fakeRecoverySessionLock struct {
	owners map[string]repository.SessionLockOwner
}

func (f *fakeRecoverySessionLock) Inspect(ctx context.Context, sessionID string) (repository.SessionLockOwner, error) {
	if f.owners == nil {
		return repository.SessionLockOwner{}, nil
	}
	return f.owners[sessionID], nil
}
