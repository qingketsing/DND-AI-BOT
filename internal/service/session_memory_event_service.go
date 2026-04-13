package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SessionMemoryEventService 负责将高价值业务事件转换为长期记忆更新。
type SessionMemoryEventService struct {
	memory *SessionMemoryService
}

// NewSessionMemoryEventService 创建会话记忆事件服务。
func NewSessionMemoryEventService(memory *SessionMemoryService) *SessionMemoryEventService {
	return &SessionMemoryEventService{memory: memory}
}

// RecordQuestCreated 记录任务创建事件，并将任务标题提升为当前目标。
func (s *SessionMemoryEventService) RecordQuestCreated(ctx context.Context, sessionID string, title string, summary string, now time.Time) error {
	if s == nil || s.memory == nil {
		return nil
	}

	event := fmt.Sprintf("已接任务：%s。", strings.TrimSpace(title))
	if note := strings.TrimSpace(summary); note != "" {
		event = event + " " + note
	}

	return s.RecordObjectiveChanged(ctx, sessionID, buildQuestObjective(title, summary), event, now)
}

// RecordQuestUpdated 记录任务更新事件；若任务仍处于 active，则继续刷新当前目标。
func (s *SessionMemoryEventService) RecordQuestUpdated(ctx context.Context, sessionID string, title string, status string, progressNote string, now time.Time) error {
	if s == nil || s.memory == nil {
		return nil
	}

	input := UpdateSessionMemoryInput{
		SessionID:   strings.TrimSpace(sessionID),
		AppendEvent: fmt.Sprintf("任务更新：%s（%s）。", strings.TrimSpace(title), strings.TrimSpace(status)),
	}
	if note := strings.TrimSpace(progressNote); note != "" {
		input.AppendEvent = input.AppendEvent + " " + note
	}
	if strings.TrimSpace(status) == "active" {
		return s.RecordObjectiveChanged(ctx, sessionID, buildQuestObjective(title, progressNote), input.AppendEvent, now)
	}

	_, err := s.memory.Update(ctx, input, now)
	return err
}

// RecordQuestCompleted 记录任务完成事件，并将当前目标切回待玩家决定下一步行动。
func (s *SessionMemoryEventService) RecordQuestCompleted(ctx context.Context, sessionID string, title string, completionNote string, now time.Time) error {
	if s == nil || s.memory == nil {
		return nil
	}

	event := fmt.Sprintf("任务完成：%s。", strings.TrimSpace(title))
	if note := strings.TrimSpace(completionNote); note != "" {
		event = event + " " + note
	}

	return s.RecordObjectiveChanged(ctx, sessionID, "等待下一步行动", event, now)
}

// RecordObjectiveChanged 显式记录当前目标变化。
func (s *SessionMemoryEventService) RecordObjectiveChanged(ctx context.Context, sessionID string, objective string, reason string, now time.Time) error {
	if s == nil || s.memory == nil {
		return nil
	}

	event := ""
	if reason = strings.TrimSpace(reason); reason != "" {
		if strings.HasPrefix(reason, "目标更新：") {
			event = reason
		} else {
			event = "目标更新：" + reason
		}
	}
	_, err := s.memory.Update(ctx, UpdateSessionMemoryInput{
		SessionID:        strings.TrimSpace(sessionID),
		CurrentObjective: strings.TrimSpace(objective),
		AppendEvent:      event,
	}, now)
	return err
}

// RecordSceneFact 将关键场景事实写入长期记忆。
func (s *SessionMemoryEventService) RecordSceneFact(ctx context.Context, sessionID string, sceneSummary string, fact string, now time.Time) error {
	if s == nil || s.memory == nil {
		return nil
	}

	event := strings.TrimSpace(fact)
	if event != "" && !strings.HasPrefix(event, "场景事实：") {
		event = "场景事实：" + event
	}
	_, err := s.memory.Update(ctx, UpdateSessionMemoryInput{
		SessionID:    strings.TrimSpace(sessionID),
		SceneSummary: strings.TrimSpace(sceneSummary),
		AppendEvent:  event,
	}, now)
	return err
}

func buildQuestObjective(title string, summary string) string {
	if note := strings.TrimSpace(summary); note != "" {
		return note
	}
	return strings.TrimSpace(title)
}
