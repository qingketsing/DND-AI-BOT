package soak

import "fmt"

func buildPlayerSystemPrompt() string {
	return "你是 DND 长会话测试中的玩家模拟器。你只输出下一轮玩家要发送的一句话，不要解释。"
}

func buildPlayerUserPrompt(input PlayerInput) string {
	return fmt.Sprintf(
		"场景：%s\n目标：%s\n当前轮次：%d\n最近对话：%s\n请生成下一轮真实玩家输入。",
		input.Scenario.Name,
		playerObjective(input),
		input.Round,
		formatRecentTurns(playerTurns(input), 6),
	)
}

func buildJudgeSystemPrompt() string {
	return `你是 DND AI Bot 的自动评测器。你只评估本轮回复是否成功，不要继续扮演 DM。

成功标准：
1. 回复必须回应用户本轮输入。
2. 不得忘记已建立角色、场景、任务、战斗状态。
3. 不得要求用户重复已经提供的信息。
4. 如果处于战斗，攻击、伤害、回合必须推进。
5. 如果工具或模型失败，回复必须可恢复，不能暴露无意义错误。
6. 不得凭空创造与当前状态冲突的事实。
7. 如果用户问状态，必须给出明确状态，而不是泛泛建议。

只返回 JSON：{"success":true,"score":0.9,"failure_reasons":[],"comment":"..."}`
}

func buildJudgeUserPrompt(input JudgeInput) string {
	return fmt.Sprintf(
		"场景：%s\n目标：%s\n轮次：%d\nHTTP状态：%d\n延迟ms：%d\n最近对话：%s\n本轮用户输入：%s\n本轮DM回复：%s",
		input.Scenario.Name,
		input.Scenario.Objective,
		input.Round,
		input.HTTPStatus,
		input.LatencyMS,
		formatRecentTurns(judgeTurns(input), 8),
		input.UserInput,
		input.AgentReply,
	)
}

func playerObjective(input PlayerInput) string {
	if input.GameObjective != "" {
		return input.GameObjective
	}
	return input.Scenario.Objective
}

func playerTurns(input PlayerInput) []RoundRecord {
	if input.Records != nil {
		return input.Records
	}
	return input.PreviousTurns
}

func judgeTurns(input JudgeInput) []RoundRecord {
	if input.Records != nil {
		return input.Records
	}
	return input.PreviousTurns
}

func formatRecentTurns(records []RoundRecord, limit int) string {
	if len(records) == 0 {
		return "无"
	}
	start := len(records) - limit
	if start < 0 {
		start = 0
	}
	text := ""
	for _, record := range records[start:] {
		text += fmt.Sprintf("\n第%d轮 用户：%s\nDM：%s\n成功：%t", record.Round, record.UserInput, record.AgentReply, record.Success)
	}
	return text
}
