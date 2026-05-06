package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestPGMessageJobStoreCreateAndGetByID(t *testing.T) {
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

func TestPGMessageJobStoreGetByMessageID(t *testing.T) {
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

func TestPGMessageJobStoreStatusTransitions(t *testing.T) {
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

func TestPGMessageJobStoreFailureTransitions(t *testing.T) {
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

	if err := store.MarkFailed(context.Background(), job.ID, finishedAt.Add(5*time.Second), "fatal_error", "permanent failure"); err != nil {
		t.Fatalf("expected failed to succeed, got %v", err)
	}

	failed, err := store.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("expected get after failed to succeed, got %v", err)
	}
	if failed.Status != model.MessageJobFailed || failed.LastErrorCode != "fatal_error" {
		t.Fatalf("unexpected failed state: %+v", failed)
	}
}

func TestPGMessageJobStoreReturnsNotFound(t *testing.T) {
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
