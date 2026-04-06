package tools

import (
	"bytes"
	"encoding/json"

	"DND-AI-BOT/internal/game/rules"
)

// decodeToolInput 将工具原始 JSON 输入解析到目标结构中。
func decodeToolInput(raw json.RawMessage, target any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		trimmed = []byte("{}")
	}
	if err := json.Unmarshal(trimmed, target); err != nil {
		return ErrInvalidToolInput
	}
	return nil
}

// newToolOutput 构造统一的工具输出结构。
func newToolOutput(toolName string, content any) CallOutput {
	return CallOutput{
		ToolName: toolName,
		Content:  content,
	}
}

// objectSchema 构造最小对象输入 schema。
func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// parseRollMode 将字符串模式转换为规则引擎使用的掷骰模式。
func parseRollMode(mode string) rules.RollMode {
	if mode == "" {
		return rules.RollModeNormal
	}
	return rules.RollMode(mode)
}
