package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/repository/memory"
)

func TestAsyncMessageServiceEnqueueMessagePersistsMessageJobAndOutboxTransactionally(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	sessions := &fakeAsyncEnqueueSessionRepository{session: session}
	jobs := newFakeAsyncMessageJobRepository()
	service := NewAsyncMessageService(sessions, jobs)

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
	if sessions.enqueueCalls != 1 {
		t.Fatalf("expected one transactional enqueue call, got %d", sessions.enqueueCalls)
	}
	if sessions.saveCalls != 0 {
		t.Fatalf("expected direct session save to be skipped, got %d calls", sessions.saveCalls)
	}
	if jobs.createCalls != 0 {
		t.Fatalf("expected direct job create to be skipped, got %d calls", jobs.createCalls)
	}
	if sessions.lastSession == nil {
		t.Fatal("expected transactional session to be persisted")
	}
	if len(sessions.lastSession.History) != 1 {
		t.Fatalf("expected one persisted history record, got %d", len(sessions.lastSession.History))
	}
	if sessions.lastSession.History[0].ID != result.MessageID {
		t.Fatalf("expected persisted message id %q, got %q", result.MessageID, sessions.lastSession.History[0].ID)
	}
	if sessions.lastJob.ID != result.JobID {
		t.Fatalf("expected persisted job id %q, got %q", result.JobID, sessions.lastJob.ID)
	}
	if sessions.lastJob.Status != model.MessageJobQueued {
		t.Fatalf("expected queued job status, got %q", sessions.lastJob.Status)
	}
	if sessions.lastEvent.Status != model.OutboxEventPending {
		t.Fatalf("expected pending outbox status, got %q", sessions.lastEvent.Status)
	}
	if sessions.lastEvent.AggregateID != result.JobID {
		t.Fatalf("expected outbox aggregate id %q, got %q", result.JobID, sessions.lastEvent.AggregateID)
	}

	var payload map[string]any
	if err := json.Unmarshal(sessions.lastEvent.PayloadJSON, &payload); err != nil {
		t.Fatalf("expected valid outbox payload, got %v", err)
	}
	if payload["job_id"] != result.JobID {
		t.Fatalf("expected payload job_id %q, got %v", result.JobID, payload["job_id"])
	}
	if payload["message_id"] != result.MessageID {
		t.Fatalf("expected payload message_id %q, got %v", result.MessageID, payload["message_id"])
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("expected payload session_id session-1, got %v", payload["session_id"])
	}
	if payload["user_id"] != "user-1" {
		t.Fatalf("expected payload user_id user-1, got %v", payload["user_id"])
	}
}

func TestAsyncMessageServiceEnqueueMessagePersistsQueuedJobWithMemoryRepositories(t *testing.T) {
	sessions, jobs := memory.NewAsyncMessageRepositories()
	service := NewAsyncMessageService(sessions, jobs)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	result, err := service.EnqueueMessage(context.Background(), "user-1", "Alice", EnqueueMessageInput{
		SessionID: "session-1",
		Content:   "hello async",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected enqueue message to succeed, got %v", err)
	}

	job, err := jobs.GetByID(context.Background(), result.JobID)
	if err != nil {
		t.Fatalf("expected queued job to be queryable, got %v", err)
	}
	if job.MessageID != result.MessageID {
		t.Fatalf("expected job message id %q, got %q", result.MessageID, job.MessageID)
	}
	if job.Status != model.MessageJobQueued {
		t.Fatalf("expected queued job status, got %q", job.Status)
	}
}

func TestAsyncMessageServiceGetMessageStatusDoesNotInferAssistantReplyFromNextAgentMessage(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	service := NewAsyncMessageService(sessions, jobs)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	userRecord := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	session.AppendAgentMessage(model.SessionUser{ID: "agent", Name: "DM Agent"}, "unrelated reply", now.Add(2*time.Minute))
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

	if result.AssistantReply != nil {
		t.Fatalf("expected assistant reply to be absent without explicit link, got %+v", result.AssistantReply)
	}
}

func TestAsyncMessageServiceGetMessageStatusReturnsExplicitAssistantReply(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	service := NewAsyncMessageService(sessions, jobs)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	userRecord := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	replyRecord := session.AppendAssistantReply(
		model.SessionUser{ID: "agent", Name: "DM Agent"},
		"world",
		userRecord.ID,
		"job-1",
		now.Add(2*time.Minute),
	)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}

	finishedAt := now.Add(4 * time.Minute)
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:           "job-1",
		MessageID:    userRecord.ID,
		SessionID:    "session-1",
		UserID:       "user-1",
		Status:       model.MessageJobCompleted,
		AttemptCount: 1,
		QueuedAt:     now.Add(time.Minute),
		FinishedAt:   &finishedAt,
		CreatedAt:    now.Add(time.Minute),
		UpdatedAt:    finishedAt,
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	result, err := service.GetMessageStatus(context.Background(), userRecord.ID, "user-1")
	if err != nil {
		t.Fatalf("expected get message status to succeed, got %v", err)
	}

	if result.AssistantReply == nil {
		t.Fatal("expected assistant reply to be present")
	}
	if result.AssistantReply.MessageID != replyRecord.ID {
		t.Fatalf("expected assistant reply id %q, got %q", replyRecord.ID, result.AssistantReply.MessageID)
	}
	if result.AssistantReply.ReplyToMessageID != userRecord.ID {
		t.Fatalf("expected reply_to_message_id %q, got %q", userRecord.ID, result.AssistantReply.ReplyToMessageID)
	}
	if result.AssistantReply.SourceJobID != "job-1" {
		t.Fatalf("expected source_job_id job-1, got %q", result.AssistantReply.SourceJobID)
	}
}

func TestAsyncMessageServiceAssistantReplyResultExposesExplicitAssociationFields(t *testing.T) {
	replyType := reflect.TypeOf(AssistantReplyResult{})
	for _, fieldName := range []string{"ReplyToMessageID", "SourceJobID"} {
		field, ok := replyType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("expected AssistantReplyResult to expose %s", fieldName)
		}
		if field.Type.Kind() != reflect.String {
			t.Fatalf("expected AssistantReplyResult.%s to be a string, got %s", fieldName, field.Type)
		}
	}
}

func TestAsyncMessageServiceGetMessageStatusReturnsForbiddenForDifferentOwner(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	service := NewAsyncMessageService(sessions, jobs)

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

type fakeAsyncEnqueueSessionRepository struct {
	session      *model.Session
	saveCalls    int
	enqueueCalls int
	lastSession  *model.Session
	lastJob      model.MessageJob
	lastEvent    model.OutboxEvent
}

func (f *fakeAsyncEnqueueSessionRepository) Save(ctx context.Context, session *model.Session) error {
	_ = ctx
	f.saveCalls++
	f.session = cloneSession(session)
	return nil
}

func (f *fakeAsyncEnqueueSessionRepository) Load(ctx context.Context, sessionID string) (*model.Session, error) {
	_ = ctx
	if f.session == nil || f.session.ID != sessionID {
		return nil, repository.ErrSessionNotFound
	}
	return cloneSession(f.session), nil
}

func (f *fakeAsyncEnqueueSessionRepository) ListByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	_ = ctx
	if f.session == nil || f.session.UserID != userID {
		return nil, nil
	}
	return []*model.Session{cloneSession(f.session)}, nil
}

func (f *fakeAsyncEnqueueSessionRepository) Delete(ctx context.Context, sessionID string) error {
	_ = ctx
	_ = sessionID
	return nil
}

func (f *fakeAsyncEnqueueSessionRepository) EnqueueAsyncMessage(ctx context.Context, session *model.Session, job model.MessageJob, event model.OutboxEvent) error {
	_ = ctx
	f.enqueueCalls++
	f.session = cloneSession(session)
	f.lastSession = cloneSession(session)
	f.lastJob = job
	f.lastEvent = event
	return nil
}

type fakeAsyncMessageJobRepository struct {
	createCalls int
	jobs        map[string]model.MessageJob
	messageToID map[string]string
}

func newFakeAsyncMessageJobRepository() *fakeAsyncMessageJobRepository {
	return &fakeAsyncMessageJobRepository{
		jobs:        make(map[string]model.MessageJob),
		messageToID: make(map[string]string),
	}
}

func (f *fakeAsyncMessageJobRepository) Create(ctx context.Context, job model.MessageJob) error {
	_ = ctx
	f.createCalls++
	f.jobs[job.ID] = job
	f.messageToID[job.MessageID] = job.ID
	return nil
}

func (f *fakeAsyncMessageJobRepository) GetByID(ctx context.Context, jobID string) (*model.MessageJob, error) {
	_ = ctx
	job, ok := f.jobs[jobID]
	if !ok {
		return nil, repository.ErrMessageJobNotFound
	}
	cloned := job
	return &cloned, nil
}

func (f *fakeAsyncMessageJobRepository) GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error) {
	_ = ctx
	jobID, ok := f.messageToID[messageID]
	if !ok {
		return nil, repository.ErrMessageJobNotFound
	}
	return f.GetByID(ctx, jobID)
}

func (f *fakeAsyncMessageJobRepository) MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error {
	_ = ctx
	_ = jobID
	_ = workerID
	_ = startedAt
	return nil
}

func (f *fakeAsyncMessageJobRepository) MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error {
	_ = ctx
	_ = jobID
	_ = finishedAt
	_ = latencyMS
	return nil
}

func (f *fakeAsyncMessageJobRepository) MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	_ = ctx
	_ = jobID
	_ = finishedAt
	_ = errorCode
	_ = errorMessage
	return nil
}

func (f *fakeAsyncMessageJobRepository) MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	_ = ctx
	_ = jobID
	_ = finishedAt
	_ = errorCode
	_ = errorMessage
	return nil
}

func (f *fakeAsyncMessageJobRepository) IncrementAttempt(ctx context.Context, jobID string) error {
	_ = ctx
	_ = jobID
	return nil
}

func cloneSession(session *model.Session) *model.Session {
	if session == nil {
		return nil
	}
	cloned := model.RestoreSession(session.ToSnapshot())
	return cloned
}
