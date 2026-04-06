package context

import (
	"context"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

// SessionContextStore 定义通用会话上下文读取接口。
type SessionContextStore interface {
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	GetRecentRecords(ctx context.Context, sessionID string, limit int) ([]model.HistoryRecord, error)
	GetLastRecord(ctx context.Context, sessionID string) (model.HistoryRecord, bool, error)
	GetChannel(ctx context.Context, sessionID string) (model.Channel, error)
}

// DefaultSessionContextStore 基于会话仓库实现通用上下文读取。
type DefaultSessionContextStore struct {
	repository repository.SessionRepository
}

// NewSessionContextStore 创建默认会话上下文读取实现。
func NewSessionContextStore(repository repository.SessionRepository) *DefaultSessionContextStore {
	return &DefaultSessionContextStore{repository: repository}
}

// GetSession 读取完整会话。
func (s *DefaultSessionContextStore) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return s.repository.Load(ctx, sessionID)
}

// GetRecentRecords 返回最近的 limit 条历史记录。
func (s *DefaultSessionContextStore) GetRecentRecords(ctx context.Context, sessionID string, limit int) ([]model.HistoryRecord, error) {
	session, err := s.repository.Load(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		return []model.HistoryRecord{}, nil
	}

	history := session.HistoryRecords()
	if limit >= len(history) {
		return history, nil
	}

	return history[len(history)-limit:], nil
}

// GetLastRecord 返回会话中的最后一条记录。
func (s *DefaultSessionContextStore) GetLastRecord(ctx context.Context, sessionID string) (model.HistoryRecord, bool, error) {
	session, err := s.repository.Load(ctx, sessionID)
	if err != nil {
		return model.HistoryRecord{}, false, err
	}

	record, ok := session.LastRecord()
	return record, ok, nil
}

// GetChannel 返回会话绑定的渠道。
func (s *DefaultSessionContextStore) GetChannel(ctx context.Context, sessionID string) (model.Channel, error) {
	session, err := s.repository.Load(ctx, sessionID)
	if err != nil {
		return "", err
	}

	return session.Channel, nil
}
