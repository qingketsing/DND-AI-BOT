package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentprompt "DND-AI-BOT/internal/agent/prompt"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
)

var ErrSessionBusy = errors.New("session is already processing another message")

const defaultSessionLockTTL = 5 * time.Minute

// MessageJobProcessor 处理单个异步消息任务。
type MessageJobProcessor struct {
	sessions     repository.SessionRepository
	jobs         repository.MessageJobRepository
	lock         SessionLock
	agentService *service.AgentService
	now          func() time.Time
	workerID     string
}

// MessageJobProcessorOption 定义处理器可选配置。
type MessageJobProcessorOption func(*MessageJobProcessor)

// WithMessageJobProcessorClock 注入测试时钟。
func WithMessageJobProcessorClock(now func() time.Time) MessageJobProcessorOption {
	return func(processor *MessageJobProcessor) {
		if now != nil {
			processor.now = now
		}
	}
}

// WithMessageJobProcessorWorkerID 注入 worker 标识。
func WithMessageJobProcessorWorkerID(workerID string) MessageJobProcessorOption {
	return func(processor *MessageJobProcessor) {
		if workerID != "" {
			processor.workerID = workerID
		}
	}
}

// NewMessageJobProcessor 创建异步消息任务处理器。
func NewMessageJobProcessor(
	sessions repository.SessionRepository,
	jobs repository.MessageJobRepository,
	lock SessionLock,
	agentService *service.AgentService,
	options ...MessageJobProcessorOption,
) *MessageJobProcessor {
	processor := &MessageJobProcessor{
		sessions:     sessions,
		jobs:         jobs,
		lock:         lock,
		agentService: agentService,
		now:          func() time.Time { return time.Now().UTC() },
		workerID:     "worker-default",
	}
	for _, option := range options {
		if option != nil {
			option(processor)
		}
	}
	return processor
}

// ProcessMessageJob 执行单条异步消息任务。
func (p *MessageJobProcessor) ProcessMessageJob(ctx context.Context, payload queue.MessageJobPayload) error {
	if p.jobs == nil || p.sessions == nil || p.lock == nil || p.agentService == nil {
		return errors.New("message job processor is not fully configured")
	}

	job, err := p.jobs.GetByID(ctx, payload.JobID)
	if err != nil {
		return err
	}
	if job.Status == model.MessageJobCompleted {
		return nil
	}

	locked, err := p.lock.Acquire(ctx, payload.SessionID, payload.JobID, p.workerID, defaultSessionLockTTL)
	if err != nil {
		return err
	}
	if !locked {
		return ErrSessionBusy
	}
	defer func() {
		_ = p.lock.Release(context.Background(), payload.SessionID, payload.JobID, p.workerID)
	}()

	startedAt := p.now()
	if err := p.jobs.IncrementAttempt(ctx, payload.JobID); err != nil {
		return err
	}
	if err := p.jobs.MarkProcessing(ctx, payload.JobID, p.workerID, startedAt); err != nil {
		return err
	}

	session, err := p.sessions.Load(ctx, payload.SessionID)
	if err != nil {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_load_failed", err.Error())
		return err
	}
	userMessage, ok := findUserMessageByID(*session, payload.MessageID)
	if !ok {
		err = fmt.Errorf("message %s not found in session %s", payload.MessageID, payload.SessionID)
		_ = p.jobs.MarkFailed(ctx, payload.JobID, p.now(), "message_not_found", err.Error())
		return err
	}

	reply, err := p.agentService.Reply(ctx, service.AgentReplyInput{
		SessionID:    payload.SessionID,
		SystemPrompt: agentprompt.DefaultSystemPrompt,
		UserMessage:  userMessage.Message.Content,
	})
	if err != nil {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "agent_reply_failed", err.Error())
		return err
	}

	session.AppendAssistantReply(
		model.SessionUser{ID: "agent", Name: "DM Agent"},
		reply.Reply,
		payload.MessageID,
		payload.JobID,
		p.now(),
	)
	if err := p.sessions.Save(ctx, session); err != nil {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_save_failed", err.Error())
		return err
	}

	finishedAt := p.now()
	return p.jobs.MarkCompleted(ctx, payload.JobID, finishedAt, finishedAt.Sub(startedAt).Milliseconds())
}

func findUserMessageByID(session model.Session, messageID string) (model.HistoryRecord, bool) {
	for _, record := range session.History {
		if record.ID != messageID {
			continue
		}
		if record.Source != model.MessageSourceUser {
			return model.HistoryRecord{}, false
		}
		return record, true
	}
	return model.HistoryRecord{}, false
}
