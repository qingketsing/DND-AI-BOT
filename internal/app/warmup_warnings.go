package app

import (
	"fmt"
	"strings"

	"DND-AI-BOT/internal/service"
)

func composeWarmupWarningsPrompt(warnings []service.WarmupWarning) string {
	lines := make([]string, 0, len(warnings)+1)
	for _, warning := range warnings {
		if warning.Err == nil {
			continue
		}
		source := strings.TrimSpace(warning.Source)
		if source == "" {
			source = "unknown"
		}
		lines = append(lines, fmt.Sprintf("- %s: %v", source, warning.Err))
	}
	if len(lines) == 0 {
		return ""
	}
	return "知识库预热降级：\n知识库预热部分失败。回答时可以基于已确认上下文继续，但涉及具体规则或设定细节时，需要说明可能需要稍后重新检索确认。\n" + strings.Join(lines, "\n")
}
