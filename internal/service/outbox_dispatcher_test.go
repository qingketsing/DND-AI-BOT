package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
)

func TestOutboxDispatcherPublishesPendingEventsAndMarksState(t *testing.T) {
	now := time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC)
	payload := queue.MessageJobPayload{
		JobID:     "job-1",
		MessageID: "msg-1",
		SessionID: "session-1",
		UserID:    "user-1",
		Attempt:   1,
		QueuedAt:  now,
	}
	raw, err := queue.EncodeMessageJobPayload(payload)
	if err != nil {
		t.Fatalf("expected payload encode to succeed, got %v", err)
	}

	outbox := &fakeOutboxEventRepository{
		events: []model.OutboxEvent{{
			ID:            "outbox-1",
			AggregateType: "message_job",
			AggregateID:   "job-1",
			EventType:     "message_job_queued",
			PayloadJSON:   raw,
			Status:        model.OutboxEventPending,
			CreatedAt:     now,
			UpdatedAt:     now,
		}},
	}
	jobs := &fakeDispatcherMessageJobRepository{}
	publisher := &fakeDispatcherPublisher{}
	dispatcher := NewOutboxDispatcher(outbox, jobs, publisher, 10)

	dispatched, err := dispatcher.DispatchOnce(context.Background(), now)
	if err != nil {
		t.Fatalf("expected dispatch to succeed, got %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("expected 1 dispatched event, got %d", dispatched)
	}
	if len(publisher.payloads) != 1 || publisher.payloads[0] != payload {
		t.Fatalf("expected payload %+v, got %+v", payload, publisher.payloads)
	}
	if len(outbox.publishedIDs) != 1 || outbox.publishedIDs[0] != "outbox-1" {
		t.Fatalf("expected outbox-1 to be marked published, got %+v", outbox.publishedIDs)
	}
	if len(jobs.publishedJobIDs) != 1 || jobs.publishedJobIDs[0] != "job-1" {
		t.Fatalf("expected job-1 to be marked published, got %+v", jobs.publishedJobIDs)
	}
}

func TestOutboxDispatcherMarksFailedAttemptWhenPublishFails(t *testing.T) {
	now := time.Date(2026, 5, 10, 11, 10, 0, 0, time.UTC)
	payload := queue.MessageJobPayload{
		JobID:     "job-2",
		MessageID: "msg-2",
		SessionID: "session-2",
		UserID:    "user-2",
		Attempt:   1,
		QueuedAt:  now,
	}
	raw, err := queue.EncodeMessageJobPayload(payload)
	if err != nil {
		t.Fatalf("expected payload encode to succeed, got %v", err)
	}

	outbox := &fakeOutboxEventRepository{
		events: []model.OutboxEvent{{
			ID:            "outbox-2",
			AggregateType: "message_job",
			AggregateID:   "job-2",
			EventType:     "message_job_queued",
			PayloadJSON:   raw,
			Status:        model.OutboxEventFailed,
			CreatedAt:     now,
			UpdatedAt:     now,
		}},
	}
	jobs := &fakeDispatcherMessageJobRepository{}
	publisher := &fakeDispatcherPublisher{err: errors.New("publish failed")}
	dispatcher := NewOutboxDispatcher(outbox, jobs, publisher, 10)

	dispatched, err := dispatcher.DispatchOnce(context.Background(), now)
	if err != nil {
		t.Fatalf("expected dispatch loop to continue on publish failure, got %v", err)
	}
	if dispatched != 0 {
		t.Fatalf("expected 0 successful dispatches, got %d", dispatched)
	}
	if len(outbox.failedIDs) != 1 || outbox.failedIDs[0] != "outbox-2" {
		t.Fatalf("expected outbox-2 to be marked failed, got %+v", outbox.failedIDs)
	}
	if len(jobs.publishedJobIDs) != 0 {
		t.Fatalf("did not expect job published state update, got %+v", jobs.publishedJobIDs)
	}
	if outbox.failedErrors[0] == "" {
		t.Fatal("expected failed attempt to record last error")
	}
}

type fakeOutboxEventRepository struct {
	events        []model.OutboxEvent
	publishedIDs  []string
	failedIDs     []string
	failedErrors  []string
}

func (f *fakeOutboxEventRepository) Create(ctx context.Context, event model.OutboxEvent) error {
	_ = ctx
	f.events = append(f.events, event)
	return nil
}

func (f *fakeOutboxEventRepository) GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	_ = ctx
	if len(f.events) > limit {
		return append([]model.OutboxEvent(nil), f.events[:limit]...), nil
	}
	return append([]model.OutboxEvent(nil), f.events...), nil
}

func (f *fakeOutboxEventRepository) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	_ = ctx
	_ = publishedAt
	f.publishedIDs = append(f.publishedIDs, id)
	return nil
}

func (f *fakeOutboxEventRepository) MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, lastError string) error {
	_ = ctx
	_ = failedAt
	f.failedIDs = append(f.failedIDs, id)
	f.failedErrors = append(f.failedErrors, lastError)
	return nil
}

type fakeDispatcherMessageJobRepository struct {
	publishedJobIDs []string
}

func (f *fakeDispatcherMessageJobRepository) Create(ctx context.Context, job model.MessageJob) error {
	panic("unexpected Create call")
}

func (f *fakeDispatcherMessageJobRepository) GetByID(ctx context.Context, jobID string) (*model.MessageJob, error) {
	panic("unexpected GetByID call")
}

func (f *fakeDispatcherMessageJobRepository) GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error) {
	panic("unexpected GetByMessageID call")
}

func (f *fakeDispatcherMessageJobRepository) MarkPublished(ctx context.Context, jobID string, publishedAt time.Time) error {
	_ = ctx
	_ = publishedAt
	f.publishedJobIDs = append(f.publishedJobIDs, jobID)
	return nil
}

func (f *fakeDispatcherMessageJobRepository) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	panic("unexpected MarkProcessing call")
}

func (f *fakeDispatcherMessageJobRepository) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	panic("unexpected MarkCompleted call")
}

func (f *fakeDispatcherMessageJobRepository) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	panic("unexpected MarkRetryableFailed call")
}

func (f *fakeDispatcherMessageJobRepository) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	panic("unexpected MarkFailed call")
}

func (f *fakeDispatcherMessageJobRepository) IncrementAttempt(ctx context.Context, jobID string) error {
	panic("unexpected IncrementAttempt call")
}

type fakeDispatcherPublisher struct {
	payloads []queue.MessageJobPayload
	err      error
}

func (f *fakeDispatcherPublisher) Publish(ctx context.Context, payload queue.MessageJobPayload) error {
	_ = ctx
	if f.err != nil {
		return f.err
	}
	f.payloads = append(f.payloads, payload)
	return nil
}
