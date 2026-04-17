package prompt

import (
	"fmt"
	"strings"

	agentcontext "DND-AI-BOT/internal/agent/context"
	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
)

// ComposePreloadedSessionContextPrompt 将每轮必需的会话上下文压缩为 system prompt 片段。
func ComposePreloadedSessionContextPrompt(agentCtx agentcontext.AgentContext, gameState *state.GameState, memory *model.SessionMemory) string {
	sections := make([]string, 0, 4)

	if base := renderAgentContext(agentCtx); base != "" {
		sections = append(sections, base)
	}
	if game := renderGameStateContext(gameState); game != "" {
		sections = append(sections, game)
	}
	if mem := ComposeSessionMemoryPrompt(memory); mem != "" {
		sections = append(sections, mem)
	}
	if len(sections) == 0 {
		return ""
	}

	return "自动预加载会话上下文：\n" +
		"以下内容来自后端自动读取的最近消息、结构化游戏状态和长期记忆。回答时必须优先继承这些事实；如果用户当前输入是在补充上一轮信息，要合并上下文继续推进，不要重新追问已知字段。\n\n" +
		strings.Join(sections, "\n\n")
}

func renderAgentContext(agentCtx agentcontext.AgentContext) string {
	lines := make([]string, 0, len(agentCtx.RecentRecords)+2)
	if value := strings.TrimSpace(agentCtx.SessionID); value != "" {
		lines = append(lines, "- session_id="+value)
	}
	if value := strings.TrimSpace(string(agentCtx.Channel)); value != "" {
		lines = append(lines, "- channel="+value)
	}
	if len(agentCtx.RecentRecords) > 0 {
		lines = append(lines, "- 最近消息：")
		for _, record := range agentCtx.RecentRecords {
			content := strings.TrimSpace(record.Message.Content)
			if content == "" {
				continue
			}
			source := strings.TrimSpace(string(record.Source))
			userName := strings.TrimSpace(record.User.Name)
			if userName == "" {
				userName = strings.TrimSpace(record.User.ID)
			}
			lines = append(lines, fmt.Sprintf("  %d. [%s/%s] %s", record.Sequence, source, userName, content))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "get_agent_context：\n" + strings.Join(lines, "\n")
}

func renderGameStateContext(gameState *state.GameState) string {
	if gameState == nil {
		return ""
	}

	lines := make([]string, 0, 8)
	if value := strings.TrimSpace(gameState.CurrentScene); value != "" {
		lines = append(lines, "- 当前场景："+value)
	}
	if player := renderPlayerContext(gameState.Player); player != "" {
		lines = append(lines, player)
	}
	if draft := renderCharacterDraft(gameState.Player.Draft); draft != "" {
		lines = append(lines, draft)
	}
	if len(lines) == 0 {
		return ""
	}
	return "game_state：\n" + strings.Join(lines, "\n")
}

func renderPlayerContext(player state.PlayerState) string {
	parts := make([]string, 0, 5)
	if value := strings.TrimSpace(player.Name); value != "" {
		parts = append(parts, "name="+value)
	}
	if value := strings.TrimSpace(player.Race); value != "" {
		parts = append(parts, "race="+value)
	}
	if value := strings.TrimSpace(player.Class); value != "" {
		parts = append(parts, "class="+value)
	}
	if player.Level > 0 {
		parts = append(parts, fmt.Sprintf("level=%d", player.Level))
	}
	if len(parts) == 0 {
		return ""
	}
	return "- 已确认角色：" + strings.Join(parts, ", ")
}

func renderCharacterDraft(draft *state.CharacterDraft) string {
	if draft == nil {
		return ""
	}

	parts := make([]string, 0, 6)
	if value := strings.TrimSpace(draft.Name); value != "" {
		parts = append(parts, "name="+value)
	}
	if value := strings.TrimSpace(draft.Race); value != "" {
		parts = append(parts, "race="+value)
	}
	if value := strings.TrimSpace(draft.Class); value != "" {
		parts = append(parts, "class="+value)
	}
	if draft.Level > 0 {
		parts = append(parts, fmt.Sprintf("level=%d", draft.Level))
	}
	if value := strings.TrimSpace(draft.AbilityMethod); value != "" {
		parts = append(parts, "ability_method="+value)
	}
	if len(draft.PendingFields) > 0 {
		fields := make([]string, 0, len(draft.PendingFields))
		for _, field := range draft.PendingFields {
			if value := strings.TrimSpace(field); value != "" {
				fields = append(fields, value)
			}
		}
		if len(fields) > 0 {
			parts = append(parts, "pending_fields="+strings.Join(fields, ","))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "- 角色创建草稿：" + strings.Join(parts, ", ")
}
