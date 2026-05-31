package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestMessageJobStoreCreateAndGetByID(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)
	now := time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC)

	job := model.MessageJob{
		ID:           "job-1",
		MessageID:    "msg-1",
		SessionID:    "session-1",
		UserID:       "user-1",
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.Create(context.Background(), job); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	got, err := store.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("expected get by id to succeed, got %v", err)
	}
	if got.MessageID != job.MessageID || got.Status != model.MessageJobQueued {
		t.Fatalf("unexpected message job round trip: %+v", got)
	}
}

func TestMessageJobStoreGetByMessageID(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)
	now := time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC)

	job := model.MessageJob{
		ID:           "job-2",
		MessageID:    "msg-2",
		SessionID:    "session-2",
		UserID:       "user-2",
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.Create(context.Background(), job); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	got, err := store.GetByMessageID(context.Background(), job.MessageID)
	if err != nil {
		t.Fatalf("expected get by message id to succeed, got %v", err)
	}
	if got.ID != job.ID {
		t.Fatalf("expected job id %q, got %+v", job.ID, got)
	}
}

func TestMessageJobStoreStatusTransitions(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)
	now := time.Date(2026, 5, 6, 9, 20, 0, 0, time.UTC)

	job := model.MessageJob{
		ID:           "job-3",
		MessageID:    "msg-3",
		SessionID:    "session-3",
		UserID:       "user-3",
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.Create(context.Background(), job); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	publishedAt := now.Add(30 * time.Second)
	if err := store.MarkPublished(context.Background(), job.ID, publishedAt); err != nil {
		t.Fatalf("expected mark published to succeed, got %v", err)
	}

	startedAt := now.Add(time.Minute)
	if err := store.MarkProcessing(context.Background(), job.ID, "worker-1", startedAt); err != nil {
		t.Fatalf("expected mark processing to succeed, got %v", err)
	}
	if err := store.IncrementAttempt(context.Background(), job.ID); err != nil {
		t.Fatalf("expected increment attempt to succeed, got %v", err)
	}

	got, err := store.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("expected get after processing to succeed, got %v", err)
	}
	if got.Status != model.MessageJobProcessing || got.WorkerID != "worker-1" || got.AttemptCount != 1 {
		t.Fatalf("unexpected processing state: %+v", got)
	}

	finishedAt := startedAt.Add(15 * time.Second)
	if err := store.MarkCompleted(context.Background(), job.ID, finishedAt, 15000); err != nil {
		t.Fatalf("expected mark completed to succeed, got %v", err)
	}

	completed, err := store.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("expected get after completed to succeed, got %v", err)
	}
	if completed.Status != model.MessageJobCompleted || completed.LatencyMS != 15000 {
		t.Fatalf("unexpected completed state: %+v", completed)
	}
}

func TestMessageJobStoreRejectsInvalidStatusTransitions(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)
	now := time.Date(2026, 5, 6, 9, 25, 0, 0, time.UTC)

	job := model.MessageJob{
		ID:           "job-invalid",
		MessageID:    "msg-invalid",
		SessionID:    "session-invalid",
		UserID:       "user-invalid",
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.Create(context.Background(), job); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if err := store.MarkCompleted(context.Background(), job.ID, now.Add(time.Second), 1000); !errors.Is(err, repository.ErrMessageJobNotFound) {
		t.Fatalf("expected invalid queued->completed transition to be rejected, got %v", err)
	}

	if err := store.MarkRetryableFailed(context.Background(), job.ID, now.Add(2*time.Second), "agent_failed", "temporary"); !errors.Is(err, repository.ErrMessageJobNotFound) {
		t.Fatalf("expected invalid queued->retryable_failed transition to be rejected, got %v", err)
	}

	if err := store.MarkPublished(context.Background(), job.ID, now.Add(3*time.Second)); err != nil {
		t.Fatalf("expected queued->published to succeed, got %v", err)
	}

	if err := store.MarkPublished(context.Background(), job.ID, now.Add(4*time.Second)); !errors.Is(err, repository.ErrMessageJobNotFound) {
		t.Fatalf("expected second published transition to be rejected, got %v", err)
	}
}

func TestMessageJobStoreFailureTransitions(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)
	now := time.Date(2026, 5, 6, 9, 30, 0, 0, time.UTC)

	job := model.MessageJob{
		ID:           "job-4",
		MessageID:    "msg-4",
		SessionID:    "session-4",
		UserID:       "user-4",
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.Create(context.Background(), job); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	startedAt := now.Add(5 * time.Second)
	if err := store.MarkPublished(context.Background(), job.ID, startedAt); err != nil {
		t.Fatalf("expected mark published to succeed, got %v", err)
	}
	if err := store.MarkProcessing(context.Background(), job.ID, "worker-1", startedAt); err != nil {
		t.Fatalf("expected mark processing to succeed, got %v", err)
	}

	finishedAt := now.Add(10 * time.Second)
	if err := store.MarkRetryableFailed(context.Background(), job.ID, finishedAt, "llm_timeout", "temporary failure"); err != nil {
		t.Fatalf("expected retryable failed to succeed, got %v", err)
	}

	retryable, err := store.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("expected get after retryable failed to succeed, got %v", err)
	}
	if retryable.Status != model.MessageJobRetryableFailed || retryable.LastErrorCode != "llm_timeout" {
		t.Fatalf("unexpected retryable failed state: %+v", retryable)
	}

	job2 := model.MessageJob{
		ID:           "job-4b",
		MessageID:    "msg-4b",
		SessionID:    "session-4",
		UserID:       "user-4",
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.Create(context.Background(), job2); err != nil {
		t.Fatalf("expected second create to succeed, got %v", err)
	}
	if err := store.MarkPublished(context.Background(), job2.ID, now.Add(11*time.Second)); err != nil {
		t.Fatalf("expected second mark published to succeed, got %v", err)
	}
	if err := store.MarkProcessing(context.Background(), job2.ID, "worker-2", now.Add(12*time.Second)); err != nil {
		t.Fatalf("expected second mark processing to succeed, got %v", err)
	}
	if err := store.MarkFailed(context.Background(), job2.ID, finishedAt.Add(5*time.Second), "fatal_error", "permanent failure"); err != nil {
		t.Fatalf("expected failed to succeed, got %v", err)
	}

	failed, err := store.GetByID(context.Background(), job2.ID)
	if err != nil {
		t.Fatalf("expected get after failed to succeed, got %v", err)
	}
	if failed.Status != model.MessageJobFailed || failed.LastErrorCode != "fatal_error" {
		t.Fatalf("unexpected failed state: %+v", failed)
	}
}

func TestMessageJobStoreRecoveryQueriesAndRequeue(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	processing := model.MessageJob{
		ID:           "job-processing",
		MessageID:    "msg-processing",
		SessionID:    "session-1",
		UserID:       "user-1",
		Status:       model.MessageJobProcessing,
		AttemptCount: 1,
		MaxAttempts:  3,
		WorkerID:     "worker-old",
		QueuedAt:     now.Add(-time.Minute),
		HeartbeatAt:  timePointer(now.Add(-45 * time.Second)),
		CreatedAt:    now.Add(-time.Minute),
		UpdatedAt:    now.Add(-45 * time.Second),
	}
	if err := store.Create(context.Background(), processing); err != nil {
		t.Fatalf("expected processing create to succeed, got %v", err)
	}

	stale, err := store.ListStaleProcessing(context.Background(), now.Add(-30*time.Second), 10)
	if err != nil {
		t.Fatalf("expected stale processing query to succeed, got %v", err)
	}
	if len(stale) != 1 || stale[0].ID != processing.ID {
		t.Fatalf("expected stale processing job, got %+v", stale)
	}

	heartbeatAt := now
	if err := store.MarkHeartbeat(context.Background(), processing.ID, heartbeatAt); err != nil {
		t.Fatalf("expected heartbeat to succeed, got %v", err)
	}
	afterHeartbeat, err := store.GetByID(context.Background(), processing.ID)
	if err != nil {
		t.Fatalf("expected get after heartbeat to succeed, got %v", err)
	}
	if afterHeartbeat.HeartbeatAt == nil || !afterHeartbeat.HeartbeatAt.Equal(heartbeatAt) {
		t.Fatalf("expected heartbeat_at %s, got %+v", heartbeatAt, afterHeartbeat.HeartbeatAt)
	}

	nextRetryAt := now.Add(30 * time.Second)
	if err := store.MarkRetryScheduled(context.Background(), processing.ID, now, nextRetryAt, "llm_timeout", "temporary"); err != nil {
		t.Fatalf("expected retry schedule to succeed, got %v", err)
	}
	dueEarly, err := store.ListRetryDue(context.Background(), now.Add(29*time.Second), 10)
	if err != nil {
		t.Fatalf("expected retry due early query to succeed, got %v", err)
	}
	if len(dueEarly) != 0 {
		t.Fatalf("expected no retry due before next_retry_at, got %+v", dueEarly)
	}
	due, err := store.ListRetryDue(context.Background(), nextRetryAt, 10)
	if err != nil {
		t.Fatalf("expected retry due query to succeed, got %v", err)
	}
	if len(due) != 1 || due[0].ID != processing.ID {
		t.Fatalf("expected retry due job, got %+v", due)
	}

	event := model.OutboxEvent{
		ID:            "outbox-retry",
		AggregateType: "message_job",
		AggregateID:   processing.ID,
		EventType:     "message_job_queued",
		PayloadJSON:   []byte(`{"job_id":"job-processing"}`),
		Status:        model.OutboxEventPending,
		CreatedAt:     nextRetryAt,
		UpdatedAt:     nextRetryAt,
	}
	if err := store.RequeueRetryableWithOutbox(context.Background(), due[0], event, nextRetryAt); err != nil {
		t.Fatalf("expected requeue with outbox to succeed, got %v", err)
	}
	requeued, err := store.GetByID(context.Background(), processing.ID)
	if err != nil {
		t.Fatalf("expected get requeued job to succeed, got %v", err)
	}
	if requeued.Status != model.MessageJobQueued || requeued.NextRetryAt != nil || requeued.HeartbeatAt != nil {
		t.Fatalf("unexpected requeued state: %+v", requeued)
	}
	if _, ok := state.outboxEvents[event.ID]; !ok {
		t.Fatalf("expected outbox event %s to be inserted", event.ID)
	}

	if err := store.RepairPublished(context.Background(), processing.ID, nextRetryAt.Add(time.Second)); err != nil {
		t.Fatalf("expected repair published to succeed, got %v", err)
	}
	repaired, err := store.GetByID(context.Background(), processing.ID)
	if err != nil {
		t.Fatalf("expected get repaired job to succeed, got %v", err)
	}
	if repaired.Status != model.MessageJobPublished {
		t.Fatalf("expected repaired published status, got %+v", repaired)
	}
}

func TestMessageJobStoreReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGMessageJobStore(db)

	_, err := store.GetByID(context.Background(), "missing")
	if !errors.Is(err, repository.ErrMessageJobNotFound) {
		t.Fatalf("expected ErrMessageJobNotFound, got %v", err)
	}

	_, err = store.GetByMessageID(context.Background(), "missing-message")
	if !errors.Is(err, repository.ErrMessageJobNotFound) {
		t.Fatalf("expected ErrMessageJobNotFound, got %v", err)
	}
}
