package postgres

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

func TestMigrationsAddOutboxAndAssistantMessageConstraints(t *testing.T) {
	outboxMigration := readMigrationFile(t, "013_create_outbox_events.sql")
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS outbox_events",
		"id TEXT PRIMARY KEY",
		"aggregate_type TEXT NOT NULL",
		"aggregate_id TEXT NOT NULL",
		"event_type TEXT NOT NULL",
		"payload_json JSONB NOT NULL",
		"status TEXT NOT NULL",
		"attempt_count INTEGER NOT NULL DEFAULT 0",
		"last_error TEXT NOT NULL DEFAULT ''",
		"created_at TIMESTAMPTZ NOT NULL",
		"published_at TIMESTAMPTZ NULL",
		"updated_at TIMESTAMPTZ NOT NULL",
		"CONSTRAINT chk_outbox_events_status",
		"CHECK (status IN ('pending', 'published', 'failed'))",
		"CREATE INDEX IF NOT EXISTS idx_outbox_events_status_created_at",
		"ON outbox_events(status, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate_id",
		"ON outbox_events(aggregate_id)",
	} {
		if !strings.Contains(outboxMigration, fragment) {
			t.Fatalf("expected outbox migration to contain %q", fragment)
		}
	}

	messageJobMigration := readMigrationFile(t, "012_create_message_jobs.sql")
	if !strings.Contains(messageJobMigration, "'published'") {
		t.Fatalf("expected message_jobs migration to allow published status")
	}

	messageConstraintMigration := readMigrationFile(t, "014_add_session_message_async_columns.sql")
	for _, fragment := range []string{
		"ALTER TABLE session_messages",
		"ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT ''",
		"ADD COLUMN IF NOT EXISTS source_job_id TEXT",
		"ADD COLUMN IF NOT EXISTS reply_to_message_id TEXT",
		"UPDATE session_messages",
		"WHEN 'agent' THEN 'assistant'",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_session_messages_assistant_reply_to_message_id",
		"ON session_messages(reply_to_message_id)",
		"WHERE role = 'assistant'",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_session_messages_assistant_source_job_id",
		"ON session_messages(source_job_id)",
	} {
		if !strings.Contains(messageConstraintMigration, fragment) {
			t.Fatalf("expected session_messages migration to contain %q", fragment)
		}
	}
}

func TestMigrationAddsAsyncRecoveryFields(t *testing.T) {
	recoveryMigration := readMigrationFile(t, "015_add_async_recovery_fields.sql")
	for _, fragment := range []string{
		"ALTER TABLE message_jobs",
		"ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ NULL",
		"ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ NULL",
		"CREATE INDEX IF NOT EXISTS idx_message_jobs_retryable_next_retry_at",
		"ON message_jobs(status, next_retry_at)",
		"WHERE status = 'retryable_failed'",
		"CREATE INDEX IF NOT EXISTS idx_message_jobs_processing_updated_at",
		"ON message_jobs(status, updated_at)",
		"WHERE status = 'processing'",
		"ALTER TABLE outbox_events",
		"CREATE INDEX IF NOT EXISTS idx_outbox_events_dispatch_due",
		"ON outbox_events(status, next_retry_at, created_at)",
		"WHERE status IN ('pending', 'failed')",
	} {
		if !strings.Contains(recoveryMigration, fragment) {
			t.Fatalf("expected recovery migration to contain %q", fragment)
		}
	}
}

func TestPGOutboxEventStoreCreateAndGetPending(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGOutboxEventStore(db)
	now := time.Date(2026, 5, 7, 8, 0, 0, 0, time.UTC)

	events := []model.OutboxEvent{
		{
			ID:            "event-1",
			AggregateType: "message_job",
			AggregateID:   "job-1",
			EventType:     "message_job_queued",
			PayloadJSON:   []byte(`{"job_id":"job-1"}`),
			Status:        model.OutboxEventPending,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "event-2",
			AggregateType: "message_job",
			AggregateID:   "job-2",
			EventType:     "message_job_queued",
			PayloadJSON:   []byte(`{"job_id":"job-2"}`),
			Status:        model.OutboxEventPublished,
			CreatedAt:     now.Add(time.Minute),
			PublishedAt:   timePointer(now.Add(2 * time.Minute)),
			UpdatedAt:     now.Add(2 * time.Minute),
		},
		{
			ID:            "event-3",
			AggregateType: "message_job",
			AggregateID:   "job-3",
			EventType:     "message_job_queued",
			PayloadJSON:   []byte(`{"job_id":"job-3"}`),
			Status:        model.OutboxEventPending,
			CreatedAt:     now.Add(3 * time.Minute),
			UpdatedAt:     now.Add(3 * time.Minute),
		},
		{
			ID:            "event-4",
			AggregateType: "message_job",
			AggregateID:   "job-4",
			EventType:     "message_job_queued",
			PayloadJSON:   []byte(`{"job_id":"job-4"}`),
			Status:        model.OutboxEventFailed,
			AttemptCount:  1,
			LastError:     "publish failed",
			CreatedAt:     now.Add(4 * time.Minute),
			UpdatedAt:     now.Add(4 * time.Minute),
		},
		{
			ID:            "event-5",
			AggregateType: "message_job",
			AggregateID:   "job-5",
			EventType:     "message_job_queued",
			PayloadJSON:   []byte(`{"job_id":"job-5"}`),
			Status:        model.OutboxEventPending,
			CreatedAt:     now.Add(5 * time.Minute),
			UpdatedAt:     now.Add(5 * time.Minute),
		},
	}

	for _, event := range events {
		if err := store.Create(context.Background(), event); err != nil {
			t.Fatalf("expected create to succeed, got %v", err)
		}
	}

	pending, err := store.GetPending(context.Background(), 2)
	if err != nil {
		t.Fatalf("expected get pending to succeed, got %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending events, got %d", len(pending))
	}
	if pending[0].ID != "event-1" || pending[1].ID != "event-3" {
		t.Fatalf("expected oldest pending events, got %+v", pending)
	}
	if !bytes.Equal(pending[0].PayloadJSON, []byte(`{"job_id":"job-1"}`)) {
		t.Fatalf("expected payload to round trip, got %s", string(pending[0].PayloadJSON))
	}
}

func TestPGOutboxEventStoreMarkPublished(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGOutboxEventStore(db)
	now := time.Date(2026, 5, 7, 8, 10, 0, 0, time.UTC)

	event := model.OutboxEvent{
		ID:            "event-published",
		AggregateType: "message_job",
		AggregateID:   "job-10",
		EventType:     "message_job_queued",
		PayloadJSON:   []byte(`{"job_id":"job-10"}`),
		Status:        model.OutboxEventPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.Create(context.Background(), event); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	publishedAt := now.Add(time.Minute)
	if err := store.MarkPublished(context.Background(), event.ID, publishedAt); err != nil {
		t.Fatalf("expected mark published to succeed, got %v", err)
	}

	stored := state.outboxEvents[event.ID]
	if stored.Status != model.OutboxEventPublished {
		t.Fatalf("expected published status, got %+v", stored)
	}
	if stored.PublishedAt == nil || !stored.PublishedAt.Equal(publishedAt) {
		t.Fatalf("expected published_at %s, got %+v", publishedAt, stored.PublishedAt)
	}
	if stored.UpdatedAt != publishedAt {
		t.Fatalf("expected updated_at %s, got %s", publishedAt, stored.UpdatedAt)
	}

	pending, err := store.GetPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected get pending to succeed, got %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending events after publish, got %+v", pending)
	}
}

func TestPGOutboxEventStoreMarkFailedAttempt(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGOutboxEventStore(db)
	now := time.Date(2026, 5, 7, 8, 20, 0, 0, time.UTC)

	event := model.OutboxEvent{
		ID:            "event-failed",
		AggregateType: "message_job",
		AggregateID:   "job-20",
		EventType:     "message_job_queued",
		PayloadJSON:   []byte(`{"job_id":"job-20"}`),
		Status:        model.OutboxEventPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.Create(context.Background(), event); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	failedAt := now.Add(45 * time.Second)
	nextRetryAt := failedAt.Add(30 * time.Second)
	if err := store.MarkFailedAttempt(context.Background(), event.ID, failedAt, nextRetryAt, "broker unavailable"); err != nil {
		t.Fatalf("expected mark failed attempt to succeed, got %v", err)
	}

	stored := state.outboxEvents[event.ID]
	if stored.Status != model.OutboxEventFailed {
		t.Fatalf("expected failed status, got %+v", stored)
	}
	if stored.AttemptCount != 1 {
		t.Fatalf("expected attempt_count 1, got %+v", stored)
	}
	if stored.LastError != "broker unavailable" {
		t.Fatalf("expected last_error to persist, got %+v", stored)
	}
	if stored.NextRetryAt == nil || !stored.NextRetryAt.Equal(nextRetryAt) {
		t.Fatalf("expected next_retry_at %s, got %+v", nextRetryAt, stored.NextRetryAt)
	}
	if stored.UpdatedAt != failedAt {
		t.Fatalf("expected updated_at %s, got %s", failedAt, stored.UpdatedAt)
	}

	pending, err := store.GetPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected get pending to succeed, got %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected failed event to remain retryable, got %+v", pending)
	}
	if pending[0].ID != event.ID {
		t.Fatalf("expected failed event %s to be returned, got %+v", event.ID, pending)
	}
	if pending[0].Status != model.OutboxEventFailed {
		t.Fatalf("expected returned event to retain failed status, got %+v", pending[0])
	}
}

func TestPGOutboxEventStoreSkipsFailedEventsUntilRetryTime(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGOutboxEventStore(db)
	now := time.Now().UTC()

	futureRetry := now.Add(time.Hour)
	event := model.OutboxEvent{
		ID:            "event-future-retry",
		AggregateType: "message_job",
		AggregateID:   "job-future-retry",
		EventType:     "message_job_queued",
		PayloadJSON:   []byte(`{"job_id":"job-future-retry"}`),
		Status:        model.OutboxEventFailed,
		AttemptCount:  1,
		LastError:     "broker unavailable",
		CreatedAt:     now,
		NextRetryAt:   &futureRetry,
		UpdatedAt:     now,
	}
	if err := store.Create(context.Background(), event); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	pending, err := store.GetPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected get pending to succeed, got %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected future failed event to be skipped, got %+v", pending)
	}
}

func TestPGOutboxEventStoreReturnsNotFound(t *testing.T) {
	state := newFakePGState()
	db := newFakePGDB(t, state)
	store := NewPGOutboxEventStore(db)
	now := time.Date(2026, 5, 7, 8, 30, 0, 0, time.UTC)

	if err := store.MarkPublished(context.Background(), "missing", now); !errors.Is(err, repository.ErrOutboxEventNotFound) {
		t.Fatalf("expected ErrOutboxEventNotFound, got %v", err)
	}
	if err := store.MarkFailedAttempt(context.Background(), "missing", now, now.Add(30*time.Second), "missing"); !errors.Is(err, repository.ErrOutboxEventNotFound) {
		t.Fatalf("expected ErrOutboxEventNotFound, got %v", err)
	}
}

func readMigrationFile(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
	if err != nil {
		t.Fatalf("expected migration %s to be readable, got %v", name, err)
	}
	return string(content)
}

func timePointer(value time.Time) *time.Time {
	return &value
}
