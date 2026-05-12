package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository/memory"
	"DND-AI-BOT/internal/service"
)

func TestMessageJobProcessorProcessMessageJobAppendsAssistantReplyAndCompletesJob(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	lock := &fakeSessionLock{acquireOK: true}
	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		_ = ctx
		if input.UserMessage != "hello" {
			t.Fatalf("expected user message hello, got %q", input.UserMessage)
		}
		return service.AgentReplyResult{Reply: "world"}, nil
	}, nil)

	createdAt := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tick := 0
	processor := NewMessageJobProcessor(
		sessions,
		jobs,
		lock,
		agentService,
		WithMessageJobProcessorWorkerID("worker-1"),
		WithMessageJobProcessorClock(func() time.Time {
			current := createdAt.Add(time.Duration(tick) * time.Second)
			tick++
			return current
		}),
	)

	session := model.NewSession("session-1", "user-1", model.ChannelWeb, createdAt)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", createdAt.Add(time.Minute))
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobPublished,
		QueuedAt:  createdAt.Add(time.Minute),
		CreatedAt: createdAt.Add(time.Minute),
		UpdatedAt: createdAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	err := processor.ProcessMessageJob(context.Background(), queue.MessageJobPayload{
		JobID:     "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Attempt:   1,
		QueuedAt:  createdAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("expected process message job to succeed, got %v", err)
	}

	job, err := jobs.GetByID(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("expected job lookup to succeed, got %v", err)
	}
	if job.Status != model.MessageJobCompleted {
		t.Fatalf("expected completed job, got %q", job.Status)
	}
	if job.AttemptCount != 1 {
		t.Fatalf("expected attempt count 1, got %d", job.AttemptCount)
	}
	if job.WorkerID != "worker-1" {
		t.Fatalf("expected worker-1, got %q", job.WorkerID)
	}

	loadedSession, err := sessions.Load(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected session load to succeed, got %v", err)
	}
	if len(loadedSession.History) != 2 {
		t.Fatalf("expected 2 history records, got %d", len(loadedSession.History))
	}
	if loadedSession.History[1].Source != model.MessageSourceAgent {
		t.Fatalf("expected agent message source, got %q", loadedSession.History[1].Source)
	}
	if loadedSession.History[1].Message.Content != "world" {
		t.Fatalf("expected assistant reply world, got %q", loadedSession.History[1].Message.Content)
	}
	if loadedSession.History[1].ReplyToMessageID != record.ID {
		t.Fatalf("expected reply_to_message_id %q, got %q", record.ID, loadedSession.History[1].ReplyToMessageID)
	}
	if loadedSession.History[1].SourceJobID != "job-1" {
		t.Fatalf("expected source_job_id job-1, got %q", loadedSession.History[1].SourceJobID)
	}
	if !lock.released {
		t.Fatal("expected session lock to be released")
	}
}

func TestMessageJobProcessorProcessMessageJobReturnsSessionBusyWhenLockUnavailable(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	lock := &fakeSessionLock{acquireOK: false}
	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		_ = ctx
		_ = input
		return service.AgentReplyResult{Reply: "world"}, nil
	}, nil)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	processor := NewMessageJobProcessor(sessions, jobs, lock, agentService, WithMessageJobProcessorClock(func() time.Time { return now }))
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
		Status:    model.MessageJobPublished,
		QueuedAt:  now.Add(time.Minute),
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	err := processor.ProcessMessageJob(context.Background(), queue.MessageJobPayload{
		JobID:     "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
	})
	if !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("expected ErrSessionBusy, got %v", err)
	}

	job, err := jobs.GetByID(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("expected job lookup to succeed, got %v", err)
	}
	if job.Status != model.MessageJobRetryableFailed {
		t.Fatalf("expected retryable_failed job, got %q", job.Status)
	}
	if job.LastErrorCode != "session_busy" {
		t.Fatalf("expected session_busy error code, got %q", job.LastErrorCode)
	}
}

func TestMessageJobProcessorProcessMessageJobReturnsWithoutReprocessingCompletedJob(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	lock := &fakeSessionLock{acquireOK: true}
	agentCalls := 0
	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		_ = ctx
		_ = input
		agentCalls++
		return service.AgentReplyResult{Reply: "should-not-run"}, nil
	}, nil)

	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	processor := NewMessageJobProcessor(sessions, jobs, lock, agentService, WithMessageJobProcessorClock(func() time.Time { return now }))
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobCompleted,
		QueuedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	if err := processor.ProcessMessageJob(context.Background(), queue.MessageJobPayload{
		JobID:     "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
	}); err != nil {
		t.Fatalf("expected completed job to short-circuit, got %v", err)
	}

	if agentCalls != 0 {
		t.Fatalf("expected no agent calls, got %d", agentCalls)
	}
	if lock.acquireCalls != 0 {
		t.Fatalf("expected no lock acquisition, got %d", lock.acquireCalls)
	}
}

func TestMessageJobProcessorProcessMessageJobMarksRetryableFailureWhenAgentReturnsError(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	lock := &fakeSessionLock{acquireOK: true}
	agentService := service.NewAgentService(nil, nil)

	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	processor := NewMessageJobProcessor(sessions, jobs, lock, agentService, WithMessageJobProcessorClock(func() time.Time { return now }))
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
		Status:    model.MessageJobPublished,
		QueuedAt:  now.Add(time.Minute),
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	err := processor.ProcessMessageJob(context.Background(), queue.MessageJobPayload{
		JobID:     "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
	})
	if !errors.Is(err, service.ErrInvalidAgentService) {
		t.Fatalf("expected ErrInvalidAgentService, got %v", err)
	}

	job, err := jobs.GetByID(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("expected job lookup to succeed, got %v", err)
	}
	if job.Status != model.MessageJobRetryableFailed {
		t.Fatalf("expected retryable failed job, got %q", job.Status)
	}
	if job.LastErrorCode != "agent_reply_failed" {
		t.Fatalf("expected agent_reply_failed code, got %q", job.LastErrorCode)
	}
}

func TestMessageJobProcessorProcessMessageJobStopsWhenHeartbeatRenewFails(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	lock := &fakeSessionLock{
		acquireOK:       true,
		renewErr:        errors.New("renew failed"),
		heartbeatTicker: make(chan time.Time, 1),
	}
	agentStarted := make(chan struct{}, 1)
	releaseAgent := make(chan struct{})
	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		_ = ctx
		_ = input
		agentStarted <- struct{}{}
		<-releaseAgent
		return service.AgentReplyResult{Reply: "world"}, nil
	}, nil)

	now := time.Date(2026, 5, 12, 13, 0, 0, 0, time.UTC)
	processor := NewMessageJobProcessor(
		sessions,
		jobs,
		lock,
		agentService,
		WithMessageJobProcessorClock(func() time.Time { return now }),
		WithMessageJobProcessorHeartbeatInterval(time.Second),
		WithMessageJobProcessorLockTTL(3*time.Minute),
		WithMessageJobProcessorHeartbeatTickerFactory(func(time.Duration) <-chan time.Time { return lock.heartbeatTicker }),
	)

	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobPublished,
		QueuedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- processor.ProcessMessageJob(context.Background(), queue.MessageJobPayload{
			JobID:     "job-1",
			MessageID: record.ID,
			SessionID: "session-1",
			UserID:    "user-1",
		})
	}()

	<-agentStarted
	lock.heartbeatTicker <- time.Now()
	close(releaseAgent)

	err := <-errCh
	if err == nil || err.Error() != "renew failed" {
		t.Fatalf("expected renew failure, got %v", err)
	}

	job, jobErr := jobs.GetByID(context.Background(), "job-1")
	if jobErr != nil {
		t.Fatalf("expected job lookup to succeed, got %v", jobErr)
	}
	if job.Status != model.MessageJobRetryableFailed {
		t.Fatalf("expected retryable_failed after renew failure, got %q", job.Status)
	}
	if job.LastErrorCode != "session_lock_renew_failed" {
		t.Fatalf("expected session_lock_renew_failed, got %q", job.LastErrorCode)
	}

	loadedSession, loadErr := sessions.Load(context.Background(), "session-1")
	if loadErr != nil {
		t.Fatalf("expected session load to succeed, got %v", loadErr)
	}
	if len(loadedSession.History) != 1 {
		t.Fatalf("expected no assistant reply to be appended, got %d records", len(loadedSession.History))
	}
}

func TestMessageJobProcessorProcessMessageJobTreatsIdempotentSaveConflictAsSuccess(t *testing.T) {
	baseSessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	lock := &fakeSessionLock{acquireOK: true}
	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		_ = ctx
		_ = input
		return service.AgentReplyResult{Reply: "world"}, nil
	}, nil)

	now := time.Date(2026, 5, 12, 14, 0, 0, 0, time.UTC)
	processor := NewMessageJobProcessor(
		&idempotentConflictSessionRepository{SessionRepository: baseSessions},
		jobs,
		lock,
		agentService,
		WithMessageJobProcessorClock(func() time.Time { return now }),
	)

	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now)
	if err := baseSessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobPublished,
		QueuedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	if err := processor.ProcessMessageJob(context.Background(), queue.MessageJobPayload{
		JobID:     "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
	}); err != nil {
		t.Fatalf("expected idempotent conflict to be treated as success, got %v", err)
	}

	job, err := jobs.GetByID(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("expected job lookup to succeed, got %v", err)
	}
	if job.Status != model.MessageJobCompleted {
		t.Fatalf("expected completed job, got %q", job.Status)
	}
}

type fakeSessionLock struct {
	acquireOK       bool
	released        bool
	acquireCalls    int
	renewCalls      int
	renewErr        error
	heartbeatTicker chan time.Time
}

func (f *fakeSessionLock) Acquire(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) (bool, error) {
	_ = ctx
	_ = sessionID
	_ = jobID
	_ = workerID
	_ = ttl
	f.acquireCalls++
	return f.acquireOK, nil
}

func (f *fakeSessionLock) Renew(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) error {
	_ = ctx
	_ = sessionID
	_ = jobID
	_ = workerID
	_ = ttl
	f.renewCalls++
	return f.renewErr
}

func (f *fakeSessionLock) Release(ctx context.Context, sessionID string, jobID string, workerID string) error {
	_ = ctx
	_ = sessionID
	_ = jobID
	_ = workerID
	f.released = true
	return nil
}

type idempotentConflictSessionRepository struct {
	*memory.SessionRepository
	saveCalls int
}

func (r *idempotentConflictSessionRepository) Save(ctx context.Context, session *model.Session) error {
	r.saveCalls++
	if r.saveCalls > 1 {
		return fakeIdempotentConflictError{}
	}
	return r.SessionRepository.Save(ctx, session)
}

type fakeIdempotentConflictError struct{}

func (fakeIdempotentConflictError) Error() string           { return "assistant reply already exists" }
func (fakeIdempotentConflictError) IdempotentSuccess() bool { return true }
