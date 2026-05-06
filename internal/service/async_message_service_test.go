package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository/memory"
)

func TestAsyncMessageServiceEnqueueMessageAppendsUserRecordAndPublishesJob(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	publisher := &fakeMessageJobPublisher{}
	service := NewAsyncMessageService(sessions, jobs, publisher)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	result, err := service.EnqueueMessage(context.Background(), "user-1", "Alice", EnqueueMessageInput{
		SessionID: "session-1",
		Content:   "  hello async  ",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected enqueue message to succeed, got %v", err)
	}

	if result.Status != "queued" {
		t.Fatalf("expected queued status, got %q", result.Status)
	}
	if len(publisher.payloads) != 1 {
		t.Fatalf("expected exactly one published payload, got %d", len(publisher.payloads))
	}
	if publisher.payloads[0].MessageID != result.MessageID {
		t.Fatalf("expected published message id %q, got %q", result.MessageID, publisher.payloads[0].MessageID)
	}

	loadedSession, err := sessions.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected session load to succeed, got %v", err)
	}
	if len(loadedSession.History) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(loadedSession.History))
	}
	if loadedSession.History[0].ID != result.MessageID {
		t.Fatalf("expected history record id %q, got %q", result.MessageID, loadedSession.History[0].ID)
	}

	job, err := jobs.GetByID(context.Background(), result.JobID)
	if err != nil {
		t.Fatalf("expected job lookup to succeed, got %v", err)
	}
	if job.MessageID != result.MessageID {
		t.Fatalf("expected job message id %q, got %q", result.MessageID, job.MessageID)
	}
	if job.Status != model.MessageJobQueued {
		t.Fatalf("expected queued job, got %q", job.Status)
	}
}

func TestAsyncMessageServiceEnqueueMessageMarksJobFailedWhenPublishFails(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	publisher := &fakeMessageJobPublisher{err: errors.New("publish failed")}
	service := NewAsyncMessageService(sessions, jobs, publisher)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	_, err := service.EnqueueMessage(context.Background(), "user-1", "Alice", EnqueueMessageInput{
		SessionID: "session-1",
		Content:   "hello async",
	}, now.Add(time.Minute))
	if err == nil {
		t.Fatal("expected enqueue message to fail when publish fails")
	}

	job, lookupErr := jobs.GetByMessageID(context.Background(), "session-1-msg-1")
	if lookupErr != nil {
		t.Fatalf("expected failed job to be stored, got %v", lookupErr)
	}
	if job.Status != model.MessageJobFailed {
		t.Fatalf("expected failed job status, got %q", job.Status)
	}
	if job.LastErrorCode != "queue_publish_failed" {
		t.Fatalf("expected queue_publish_failed error code, got %q", job.LastErrorCode)
	}
}

func TestAsyncMessageServiceGetMessageStatusReturnsAssistantReplyAfterCompletion(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	service := NewAsyncMessageService(sessions, jobs, &fakeMessageJobPublisher{})

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	userRecord := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	assistantRecord := session.AppendAgentMessage(model.SessionUser{ID: "agent", Name: "DM Agent"}, "world", now.Add(2*time.Minute))
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	startedAt := now.Add(3 * time.Minute)
	finishedAt := now.Add(4 * time.Minute)
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:           "job-1",
		MessageID:    userRecord.ID,
		SessionID:    "session-1",
		UserID:       "user-1",
		Status:       model.MessageJobCompleted,
		AttemptCount: 1,
		QueuedAt:     now.Add(time.Minute),
		StartedAt:    &startedAt,
		FinishedAt:   &finishedAt,
		LatencyMS:    60000,
		CreatedAt:    now.Add(time.Minute),
		UpdatedAt:    finishedAt,
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	result, err := service.GetMessageStatus(context.Background(), userRecord.ID, "user-1")
	if err != nil {
		t.Fatalf("expected get message status to succeed, got %v", err)
	}

	if result.Status != "completed" {
		t.Fatalf("expected completed status, got %q", result.Status)
	}
	if result.AssistantReply == nil {
		t.Fatal("expected assistant reply to be present")
	}
	if result.AssistantReply.MessageID != assistantRecord.ID {
		t.Fatalf("expected assistant reply id %q, got %q", assistantRecord.ID, result.AssistantReply.MessageID)
	}
	if result.AssistantReply.Content != "world" {
		t.Fatalf("expected assistant reply content world, got %q", result.AssistantReply.Content)
	}
}

func TestAsyncMessageServiceGetMessageStatusReturnsForbiddenForDifferentOwner(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	service := NewAsyncMessageService(sessions, jobs, &fakeMessageJobPublisher{})

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobQueued,
		QueuedAt:  now.Add(time.Minute),
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	_, err := service.GetMessageStatus(context.Background(), record.ID, "user-2")
	if !errors.Is(err, ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden, got %v", err)
	}
}

type fakeMessageJobPublisher struct {
	payloads []queue.MessageJobPayload
	err      error
}

func (f *fakeMessageJobPublisher) Publish(ctx context.Context, payload queue.MessageJobPayload) error {
	_ = ctx
	f.payloads = append(f.payloads, payload)
	return f.err
}
