package app

import (
	"context"
	"strings"

	"DND-AI-BOT/internal/agent/client"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
)

// runtimeModelSummaryAdapter 将通用模型适配器收窄为摘要器需要的调用接口。
type runtimeModelSummaryAdapter struct {
	adapter client.ModelAdapter
}

func newRuntimeModelSummaryAdapter(adapter client.ModelAdapter) *runtimeModelSummaryAdapter {
	return &runtimeModelSummaryAdapter{adapter: adapter}
}

func (m *runtimeModelSummaryAdapter) Summarize(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	output, err := m.adapter.Run(ctx, agentruntime.ModelInput{
		SessionID:    "session-summary",
		SystemPrompt: systemPrompt,
		UserMessage:  userPrompt,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output.Reply), nil
}
