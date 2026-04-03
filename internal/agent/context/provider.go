package context

import (
	basecontext "DND-AI-BOT/internal/context"
	"DND-AI-BOT/internal/model"
)

// AgentContext 表示 Agent 执行时需要的最小会话上下文。
type AgentContext struct {
	SessionID     string
	Channel       model.Channel
	RecentRecords []model.HistoryRecord
	LastRecord    *model.HistoryRecord
}

// Provider 定义面向 Agent 的上下文组装接口。
type Provider interface {
	BuildContext(sessionID string, limit int) (AgentContext, error)
}

// DefaultProvider 基于通用会话上下文接口组装 AgentContext。
type DefaultProvider struct {
	store basecontext.SessionContextStore
}

// NewProvider 创建默认 Agent 上下文提供器。
func NewProvider(store basecontext.SessionContextStore) *DefaultProvider {
	return &DefaultProvider{store: store}
}

// BuildContext 按 sessionID 组装 Agent 需要的最小上下文。
func (p *DefaultProvider) BuildContext(sessionID string, limit int) (AgentContext, error) {
	session, err := p.store.GetSession(sessionID)
	if err != nil {
		return AgentContext{}, err
	}

	recentRecords, err := p.store.GetRecentRecords(sessionID, limit)
	if err != nil {
		return AgentContext{}, err
	}

	lastRecord, ok, err := p.store.GetLastRecord(sessionID)
	if err != nil {
		return AgentContext{}, err
	}

	result := AgentContext{
		SessionID:     session.ID,
		Channel:       session.Channel,
		RecentRecords: recentRecords,
	}
	if ok {
		record := lastRecord
		result.LastRecord = &record
	}

	return result, nil
}
