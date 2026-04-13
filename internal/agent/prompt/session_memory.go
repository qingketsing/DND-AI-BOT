package prompt

import (
	"fmt"
	"strings"

	"DND-AI-BOT/internal/model"
)

// ComposeSessionMemoryPrompt 将会话长期记忆整理为紧凑的提示词片段。
func ComposeSessionMemoryPrompt(memory *model.SessionMemory) string {
	if memory == nil {
		return ""
	}

	lines := make([]string, 0, 8)
	if value := strings.TrimSpace(memory.CharacterSummary); value != "" {
		lines = append(lines, "- 角色："+value)
	}
	if value := strings.TrimSpace(memory.SceneSummary); value != "" {
		lines = append(lines, "- 场景："+value)
	}
	if value := strings.TrimSpace(memory.CurrentObjective); value != "" {
		lines = append(lines, "- 当前目标："+value)
	}
	if len(memory.RecentKeyEvents) > 0 {
		lines = append(lines, "- 最近关键事件：")
		for idx, event := range memory.RecentKeyEvents {
			event = strings.TrimSpace(event)
			if event == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("  %d. %s", idx+1, event))
		}
	}
	if len(lines) == 0 {
		return ""
	}

	return "当前会话长期记忆：\n" + strings.Join(lines, "\n")
}
