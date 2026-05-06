package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository"
)

const (
	defaultMessageJobMaxAttempts = 3
)

// AsyncMessageService 负责异步消息入队和状态查询。
type AsyncMessageService struct {
	sessions  repository.SessionRepository
	jobs      repository.MessageJobRepository
	publisher queue.MessageJobPublisher
}

// EnqueueMessageInput 定义异步消息提交所需输入。
type EnqueueMessageInput struct {
	SessionID string
	Content   string
}

// EnqueueMessageResult 定义异步消息提交的响应数据。
type EnqueueMessageResult struct {
	MessageID string
	JobID     string
	SessionID string
	Status    string
}

// MessageStatusResult 定义消息状态查询响应。
type MessageStatusResult struct {
	MessageID      string
	SessionID      string
	Status         string
	Job            MessageJobStatusResult
	AssistantReply *AssistantReplyResult
}

// MessageJobStatusResult 定义任务状态详情。
type MessageJobStatusResult struct {
	ID               string
	Status           string
	AttemptCount     int
	QueuedAt         time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	LatencyMS        int64
	LastErrorCode    string
	LastErrorMessage string
}

// AssistantReplyResult 定义异步消息的 assistant 回复。
type AssistantReplyResult struct {
	MessageID string
	Content   string
}

// NewAsyncMessageService 创建异步消息服务。
func NewAsyncMessageService(
	sessions repository.SessionRepository,
	jobs repository.MessageJobRepository,
	publisher queue.MessageJobPublisher,
) *AsyncMessageService {
	return &AsyncMessageService{
		sessions:  sessions,
		jobs:      jobs,
		publisher: publisher,
	}
}

// EnqueueMessage 校验输入、写入用户消息并提交消息任务。
func (s *AsyncMessageService) EnqueueMessage(ctx context.Context, userID string, userName string, input EnqueueMessageInput, now time.Time) (*EnqueueMessageResult, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(userName) == "" {
		return nil, ErrInvalidMessage
	}
	if s.publisher == nil {
		return nil, errors.New("message publisher is not configured")
	}

	session, err := s.getSessionForUser(ctx, userID, input.SessionID)
	if err != nil {
		return nil, err
	}

	record := session.AppendUserMessage(model.SessionUser{
		ID:   strings.TrimSpace(userID),
		Name: strings.TrimSpace(userName),
	}, content, now)
	if err := s.sessions.Save(ctx, session); err != nil {
		return nil, err
	}

	job := model.MessageJob{
		ID:           generateMessageJobID(now),
		MessageID:    record.ID,
		SessionID:    session.ID,
		UserID:       session.UserID,
		Status:       model.MessageJobQueued,
		AttemptCount: 0,
		MaxAttempts:  defaultMessageJobMaxAttempts,
		QueuedAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, err
	}

	payload := queue.MessageJobPayload{
		JobID:     job.ID,
		MessageID: job.MessageID,
		SessionID: job.SessionID,
		UserID:    job.UserID,
		Attempt:   1,
		QueuedAt:  now,
	}
	if err := s.publisher.Publish(ctx, payload); err != nil {
		_ = s.jobs.MarkFailed(ctx, job.ID, now, "queue_publish_failed", err.Error())
		return nil, err
	}

	return &EnqueueMessageResult{
		MessageID: record.ID,
		JobID:     job.ID,
		SessionID: session.ID,
		Status:    jobStatusToResponseStatus(job.Status),
	}, nil
}

// GetMessageStatus 查询异步消息状态。
func (s *AsyncMessageService) GetMessageStatus(ctx context.Context, messageID string, userID string) (*MessageStatusResult, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(messageID) == "" {
		return nil, ErrUnauthorized
	}

	job, err := s.jobs.GetByMessageID(ctx, strings.TrimSpace(messageID))
	if err != nil {
		return nil, err
	}
	if job.UserID != strings.TrimSpace(userID) {
		return nil, ErrSessionForbidden
	}

	session, err := s.sessions.Load(ctx, job.SessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != strings.TrimSpace(userID) {
		return nil, ErrSessionForbidden
	}

	return &MessageStatusResult{
		MessageID:      job.MessageID,
		SessionID:      job.SessionID,
		Status:         jobStatusToResponseStatus(job.Status),
		Job:            toMessageJobStatusResult(*job),
		AssistantReply: assistantReplyForMessage(*session, job.MessageID),
	}, nil
}

func (s *AsyncMessageService) getSessionForUser(ctx context.Context, userID string, sessionID string) (*model.Session, error) {
	session, err := s.sessions.Load(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	if session.UserID != strings.TrimSpace(userID) {
		return nil, ErrSessionForbidden
	}
	return session, nil
}

func generateMessageJobID(now time.Time) string {
	return fmt.Sprintf("job-%d", now.UnixNano())
}

func toMessageJobStatusResult(job model.MessageJob) MessageJobStatusResult {
	return MessageJobStatusResult{
		ID:               job.ID,
		Status:           jobStatusToResponseStatus(job.Status),
		AttemptCount:     job.AttemptCount,
		QueuedAt:         job.QueuedAt,
		StartedAt:        cloneTimePointer(job.StartedAt),
		FinishedAt:       cloneTimePointer(job.FinishedAt),
		LatencyMS:        job.LatencyMS,
		LastErrorCode:    job.LastErrorCode,
		LastErrorMessage: job.LastErrorMessage,
	}
}

func jobStatusToResponseStatus(status model.MessageJobStatus) string {
	switch status {
	case model.MessageJobQueued:
		return "queued"
	case model.MessageJobProcessing:
		return "processing"
	case model.MessageJobCompleted:
		return "completed"
	default:
		return "failed"
	}
}

func assistantReplyForMessage(session model.Session, messageID string) *AssistantReplyResult {
	for index, record := range session.History {
		if record.ID != messageID {
			continue
		}
		for nextIndex := index + 1; nextIndex < len(session.History); nextIndex++ {
			nextRecord := session.History[nextIndex]
			if nextRecord.Source != model.MessageSourceAgent {
				continue
			}
			return &AssistantReplyResult{
				MessageID: nextRecord.ID,
				Content:   nextRecord.Message.Content,
			}
		}
		return nil
	}
	return nil
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
