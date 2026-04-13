package service

import "context"

// SessionSummarizer 定义基于历史消息生成摘要的接口。
type SessionSummarizer interface {
	SummarizeMessages(ctx context.Context, sessionID string, messages []SummarizerMessage) (SummaryResult, error)
}

// SummarizerMessage 表示参与摘要的一条精简消息。
type SummarizerMessage struct {
	Source  string
	User    string
	Content string
}

// SummaryResult 表示摘要器输出的会话记忆结果。
type SummaryResult struct {
	CharacterSummary string
	SceneSummary     string
	CurrentObjective string
	RecentKeyEvents  []string
}
