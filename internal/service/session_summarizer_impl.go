package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SummaryModel 定义会话摘要所需的最小模型调用接口。
type SummaryModel interface {
	Summarize(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

// LLMSessionSummarizer 使用模型将旧消息压缩为结构化摘要。
type LLMSessionSummarizer struct {
	model SummaryModel
}

// NewLLMSessionSummarizer 创建基于模型的会话摘要器。
func NewLLMSessionSummarizer(model SummaryModel) *LLMSessionSummarizer {
	return &LLMSessionSummarizer{model: model}
}

// SummarizeMessages 将一段历史消息总结为结构化结果。
func (s *LLMSessionSummarizer) SummarizeMessages(ctx context.Context, sessionID string, messages []SummarizerMessage) (SummaryResult, error) {
	systemPrompt, userPrompt := buildSessionSummaryPrompt(sessionID, messages)
	raw, err := s.model.Summarize(ctx, systemPrompt, userPrompt)
	if err != nil {
		return SummaryResult{}, err
	}
	return parseSummaryResult(raw)
}

func buildSessionSummaryPrompt(sessionID string, messages []SummarizerMessage) (string, string) {
	systemPrompt := strings.TrimSpace(`你是会话摘要器。请基于给定历史消息输出严格 JSON，对话外不要输出任何解释。
JSON schema:
{
  "character_summary": "string",
  "scene_summary": "string",
  "current_objective": "string",
  "recent_key_events": ["string"]
}
要求：
- 保持简短
- recent_key_events 最多 5 条
- 不要编造历史中不存在的事实`)

	lines := make([]string, 0, len(messages)+1)
	lines = append(lines, fmt.Sprintf("session_id=%s", strings.TrimSpace(sessionID)))
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf("[%s][%s] %s", strings.TrimSpace(message.Source), strings.TrimSpace(message.User), strings.TrimSpace(message.Content)))
	}
	return systemPrompt, strings.Join(lines, "\n")
}

func parseSummaryResult(raw string) (SummaryResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed struct {
		CharacterSummary string   `json:"character_summary"`
		SceneSummary     string   `json:"scene_summary"`
		CurrentObjective string   `json:"current_objective"`
		RecentKeyEvents  []string `json:"recent_key_events"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return SummaryResult{}, err
	}

	return SummaryResult{
		CharacterSummary: strings.TrimSpace(parsed.CharacterSummary),
		SceneSummary:     strings.TrimSpace(parsed.SceneSummary),
		CurrentObjective: strings.TrimSpace(parsed.CurrentObjective),
		RecentKeyEvents:  parsed.RecentKeyEvents,
	}, nil
}
