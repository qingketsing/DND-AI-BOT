package model

// WarmupBundle 表示一轮 Agent 运行前可注入的轻量知识预热摘要。
type WarmupBundle struct {
	BaseRulesSummary      string
	BaseLoreSummary       string
	CharacterRulesSummary string
}
