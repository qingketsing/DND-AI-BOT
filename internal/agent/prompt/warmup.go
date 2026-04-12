package prompt

import (
	"strings"

	"DND-AI-BOT/internal/model"
)

// ComposeSystemPrompt 在基础系统提示词后追加轻量预热摘要。
func ComposeSystemPrompt(basePrompt string, warmup model.WarmupBundle) string {
	sections := []string{strings.TrimSpace(basePrompt)}
	warmupLines := make([]string, 0, 3)

	if strings.TrimSpace(warmup.BaseRulesSummary) != "" {
		warmupLines = append(warmupLines, "[规则摘要]\n"+strings.TrimSpace(warmup.BaseRulesSummary))
	}
	if strings.TrimSpace(warmup.BaseLoreSummary) != "" {
		warmupLines = append(warmupLines, "[设定摘要]\n"+strings.TrimSpace(warmup.BaseLoreSummary))
	}
	if strings.TrimSpace(warmup.CharacterRulesSummary) != "" {
		warmupLines = append(warmupLines, "[角色相关规则]\n"+strings.TrimSpace(warmup.CharacterRulesSummary))
	}
	if len(warmupLines) == 0 {
		return strings.TrimSpace(basePrompt)
	}

	sections = append(sections, "已知基础上下文：\n"+strings.Join(warmupLines, "\n\n"))
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}
