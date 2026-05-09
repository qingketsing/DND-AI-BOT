package memory

import (
	"context"
	"sync"

	"DND-AI-BOT/internal/model"
)

// SessionRepository 负责在内存中保存会话快照。
type SessionRepository struct {
	mu        sync.RWMutex
	sessions  map[string]model.SessionSnapshot
	asyncData *asyncMessageData
}

// NewSessionRepository 创建一个空的内存会话仓库。
func NewSessionRepository() *SessionRepository {
	return &SessionRepository{
		sessions:  make(map[string]model.SessionSnapshot),
		asyncData: newAsyncMessageData(),
	}
}

// NewAsyncMessageRepositories 创建共享异步持久化状态的 memory session/job 仓库。
func NewAsyncMessageRepositories() (*SessionRepository, *MessageJobRepository) {
	asyncData := newAsyncMessageData()
	return &SessionRepository{
			sessions:  make(map[string]model.SessionSnapshot),
			asyncData: asyncData,
		}, &MessageJobRepository{
			asyncData: asyncData,
		}
}

// Save 将会话保存为快照，已存在时直接覆盖。
func (r *SessionRepository) Save(ctx context.Context, session *model.Session) error {
	_ = ctx
	if session == nil {
		return ErrNilSession
	}
	if session.ID == "" {
		return ErrEmptySessionID
	}

	snapshot := session.ToSnapshot()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = snapshot
	return nil
}

// Load 从仓库中读取会话，并恢复为独立的领域对象。
func (r *SessionRepository) Load(ctx context.Context, id string) (*model.Session, error) {
	_ = ctx
	if id == "" {
		return nil, ErrEmptySessionID
	}

	r.mu.RLock()
	snapshot, ok := r.sessions[id]
	r.mu.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}

	return model.RestoreSession(snapshot), nil
}

// ListByUserID 返回指定用户的全部会话快照副本。
func (r *SessionRepository) ListByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*model.Session, 0)
	for _, snapshot := range r.sessions {
		if snapshot.UserID != userID {
			continue
		}
		sessions = append(sessions, model.RestoreSession(snapshot))
	}

	return sessions, nil
}

// Delete 删除指定会话。
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	_ = ctx
	if sessionID == "" {
		return ErrEmptySessionID
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}
	delete(r.sessions, sessionID)
	return nil
}

// Exists 判断目标会话是否存在于仓库中。
func (r *SessionRepository) Exists(id string) bool {
	if id == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.sessions[id]
	return ok
}

// EnqueueAsyncMessage 在内存中原子更新 session 快照并记录 outbox 事件。
func (r *SessionRepository) EnqueueAsyncMessage(ctx context.Context, session *model.Session, job model.MessageJob, event model.OutboxEvent) error {
	_ = ctx
	if session == nil {
		return ErrNilSession
	}
	if session.ID == "" {
		return ErrEmptySessionID
	}

	snapshot := session.ToSnapshot()

	r.mu.Lock()
	r.sessions[session.ID] = snapshot
	r.mu.Unlock()

	r.asyncData.mu.Lock()
	r.asyncData.jobs[job.ID] = cloneMessageJob(job)
	r.asyncData.messageToJob[job.MessageID] = job.ID
	r.asyncData.outboxEvents[event.ID] = cloneOutboxEvent(event)
	r.asyncData.mu.Unlock()
	return nil
}

type asyncMessageData struct {
	mu           sync.RWMutex
	jobs         map[string]model.MessageJob
	messageToJob map[string]string
	outboxEvents map[string]model.OutboxEvent
}

func newAsyncMessageData() *asyncMessageData {
	return &asyncMessageData{
		jobs:         make(map[string]model.MessageJob),
		messageToJob: make(map[string]string),
		outboxEvents: make(map[string]model.OutboxEvent),
	}
}

func cloneOutboxEvent(event model.OutboxEvent) model.OutboxEvent {
	cloned := event
	if event.PayloadJSON != nil {
		cloned.PayloadJSON = append([]byte(nil), event.PayloadJSON...)
	}
	if event.PublishedAt != nil {
		publishedAt := *event.PublishedAt
		cloned.PublishedAt = &publishedAt
	}
	return cloned
}
