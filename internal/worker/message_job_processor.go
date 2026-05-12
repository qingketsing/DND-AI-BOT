package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	agentprompt "DND-AI-BOT/internal/agent/prompt"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/queue"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
)

var ErrSessionBusy = errors.New("session is already processing another message")

// MessageJobProcessor 处理单个异步消息任务。
type MessageJobProcessor struct {
	sessions      repository.SessionRepository
	jobs          repository.MessageJobRepository
	lock          SessionLock
	agentService  *service.AgentService
	now           func() time.Time
	workerID      string
	lockTTL       time.Duration
	heartbeat     time.Duration
	tickerFactory func(time.Duration) <-chan time.Time
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

func WithMessageJobProcessorLockTTL(ttl time.Duration) MessageJobProcessorOption {
	return func(processor *MessageJobProcessor) {
		if ttl > 0 {
			processor.lockTTL = ttl
		}
	}
}

func WithMessageJobProcessorHeartbeatInterval(interval time.Duration) MessageJobProcessorOption {
	return func(processor *MessageJobProcessor) {
		if interval > 0 {
			processor.heartbeat = interval
		}
	}
}

func WithMessageJobProcessorHeartbeatTickerFactory(factory func(time.Duration) <-chan time.Time) MessageJobProcessorOption {
	return func(processor *MessageJobProcessor) {
		if factory != nil {
			processor.tickerFactory = factory
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
		lockTTL:      defaultSessionLockTTL,
		heartbeat:    defaultSessionLockHeartbeatInterval,
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

	locked, err := p.lock.Acquire(ctx, payload.SessionID, payload.JobID, p.workerID, p.lockTTL)
	if err != nil {
		return err
	}
	if !locked {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_busy", ErrSessionBusy.Error())
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

	processCtx, cancelProcess := context.WithCancel(ctx)
	defer cancelProcess()
	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer cancelHeartbeat()

	var (
		heartbeatErrMu sync.Mutex
		heartbeatErr   error
	)
	getHeartbeatErr := func() error {
		heartbeatErrMu.Lock()
		defer heartbeatErrMu.Unlock()
		return heartbeatErr
	}
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		for err := range startSessionLockHeartbeat(
			heartbeatCtx,
			p.lock,
			payload.SessionID,
			payload.JobID,
			p.workerID,
			p.lockTTL,
			p.heartbeat,
			p.tickerFactory,
		) {
			if err != nil {
				heartbeatErrMu.Lock()
				if heartbeatErr == nil {
					heartbeatErr = err
				}
				heartbeatErrMu.Unlock()
				cancelProcess()
				return
			}
		}
	}()
	stoppedHeartbeat := false
	stopHeartbeat := func() {
		if stoppedHeartbeat {
			return
		}
		stoppedHeartbeat = true
		cancelHeartbeat()
		<-heartbeatDone
	}
	defer stopHeartbeat()

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

	reply, err := p.agentService.Reply(processCtx, service.AgentReplyInput{
		SessionID:    payload.SessionID,
		SystemPrompt: agentprompt.DefaultSystemPrompt,
		UserMessage:  userMessage.Message.Content,
	})
	if err != nil {
		if renewErr := getHeartbeatErr(); renewErr != nil {
			_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_lock_renew_failed", renewErr.Error())
			return renewErr
		}
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "agent_reply_failed", err.Error())
		return err
	}

	if renewErr := getHeartbeatErr(); renewErr != nil {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_lock_renew_failed", renewErr.Error())
		return renewErr
	}
	if err := p.lock.Renew(ctx, payload.SessionID, payload.JobID, p.workerID, p.lockTTL); err != nil {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_lock_renew_failed", err.Error())
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
		if isIdempotentSuccessError(err) {
			finishedAt := p.now()
			stopHeartbeat()
			return p.jobs.MarkCompleted(ctx, payload.JobID, finishedAt, finishedAt.Sub(startedAt).Milliseconds())
		}
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_save_failed", err.Error())
		return err
	}

	stopHeartbeat()
	if renewErr := getHeartbeatErr(); renewErr != nil {
		_ = p.jobs.MarkRetryableFailed(ctx, payload.JobID, p.now(), "session_lock_renew_failed", renewErr.Error())
		return renewErr
	}

	finishedAt := p.now()
	return p.jobs.MarkCompleted(ctx, payload.JobID, finishedAt, finishedAt.Sub(startedAt).Milliseconds())
}

type idempotentSuccessError interface {
	IdempotentSuccess() bool
}

func isIdempotentSuccessError(err error) bool {
	var marker idempotentSuccessError
	return errors.As(err, &marker) && marker.IdempotentSuccess()
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
