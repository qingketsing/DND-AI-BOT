package tools

import (
	"context"
	"encoding/json"
	"time"
)

// ToolSpec 描述一个工具对外暴露的元信息。
type ToolSpec struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// CallInput 表示一次工具调用的统一输入。
type CallInput struct {
	SessionID string
	Raw       json.RawMessage
	Now       time.Time
}

// CallOutput 表示一次工具调用的统一输出。
type CallOutput struct {
	ToolName string
	Content  any
}

// Tool 定义所有 Agent 工具都必须满足的最小协议。
type Tool interface {
	Spec() ToolSpec
	Call(ctx context.Context, input CallInput) (CallOutput, error)
}
