package service

import (
	"context"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

// UpdateSessionMemoryInput 定义增量更新会话长期记忆所需的输入。
type UpdateSessionMemoryInput struct {
	SessionID        string
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	AppendEvent      string
}

// MergeSummaryInput 定义长度触发摘要合并所需的输入。
type MergeSummaryInput struct {
	SessionID        string
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	RecentKeyEvents  []string
}

// SessionMemoryService 负责编排会话长期记忆的读取与更新流程。
type SessionMemoryService struct {
	repository repository.SessionMemoryRepository
}

// NewSessionMemoryService 创建会话长期记忆服务。
func NewSessionMemoryService(repository repository.SessionMemoryRepository) *SessionMemoryService {
	return &SessionMemoryService{repository: repository}
}

// GetBySessionID 按会话 ID 读取长期记忆；若不存在则返回默认空记忆。
func (s *SessionMemoryService) GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error) {
	sessionID = strings.TrimSpace(sessionID)
	memory, err := s.repository.LoadBySessionID(ctx, sessionID)
	if err == nil {
		return memory, nil
	}
	if err != nil && err != repository.ErrSessionMemoryNotFound {
		return nil, err
	}
	return &model.SessionMemory{
		SessionID:       sessionID,
		RecentKeyEvents: []string{},
	}, nil
}

// Update 增量更新会话长期记忆，并在必要时追加关键事件。
func (s *SessionMemoryService) Update(ctx context.Context, input UpdateSessionMemoryInput, now time.Time) (*model.SessionMemory, error) {
	memory, err := s.GetBySessionID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}
	if value := strings.TrimSpace(input.CharacterSummary); value != "" {
		memory.CharacterSummary = value
	}
	if value := strings.TrimSpace(input.SceneSummary); value != "" {
		memory.SceneSummary = value
	}
	if value := strings.TrimSpace(input.CurrentObjective); value != "" {
		memory.CurrentObjective = value
	}
	if value := strings.TrimSpace(input.AppendEvent); value != "" {
		memory.RecentKeyEvents = append(memory.RecentKeyEvents, value)
	}
	memory.RecentKeyEvents = capRecentKeyEvents(memory.RecentKeyEvents, 5)
	memory.UpdatedAt = now
	if err := s.repository.Save(ctx, memory); err != nil {
		return nil, err
	}
	return memory, nil
}

// MergeSummary 将外部生成的摘要结果整体写回长期记忆。
func (s *SessionMemoryService) MergeSummary(ctx context.Context, input MergeSummaryInput, now time.Time) (*model.SessionMemory, error) {
	memory, err := s.GetBySessionID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}
	memory.CharacterSummary = strings.TrimSpace(input.CharacterSummary)
	memory.SceneSummary = strings.TrimSpace(input.SceneSummary)
	memory.CurrentObjective = strings.TrimSpace(input.CurrentObjective)
	memory.RecentKeyEvents = capRecentKeyEvents(append([]string(nil), input.RecentKeyEvents...), 5)
	memory.UpdatedAt = now
	if err := s.repository.Save(ctx, memory); err != nil {
		return nil, err
	}
	return memory, nil
}

func capRecentKeyEvents(events []string, max int) []string {
	if len(events) <= max {
		return events
	}
	return append([]string(nil), events[len(events)-max:]...)
}
