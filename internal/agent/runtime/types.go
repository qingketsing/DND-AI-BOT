package runtime

import (
	"context"
	"encoding/json"

	"DND-AI-BOT/internal/agent/tools"
)

// RuntimeInput 定义一轮 Agent 执行所需的输入。
type RuntimeInput struct {
	SessionID    string
	SystemPrompt string
	UserMessage  string
	MaxSteps     int
	ContextLimit int
}

// RuntimeOutput 定义一轮 Agent 执行的最终输出。
type RuntimeOutput struct {
	Reply string
	Steps []StepRecord
}

// StepRecord 表示一轮 ReAct 工具调用轨迹中的单个步骤。
type StepRecord struct {
	Thought          string          `json:"thought,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ActionName       string          `json:"action_name,omitempty"`
	ActionInput      json.RawMessage `json:"action_input,omitempty"`
	Observation      any             `json:"observation,omitempty"`
}

// ToolErrorObservation 表示工具执行失败后交还给模型的结构化观察结果。
type ToolErrorObservation struct {
	ToolName  string `json:"tool_name"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// ModelInput 定义传给模型适配层的统一输入。
type ModelInput struct {
	SessionID    string
	SystemPrompt string
	UserMessage  string
	Tools        []tools.ToolSpec
	Steps        []StepRecord
}

// ModelOutput 定义模型适配层返回给 Runtime 的统一输出。
type ModelOutput struct {
	Thought          string
	ReasoningContent string
	Reply            string
	ToolRequest      *ToolRequest
}

// ToolRequest 表示模型请求调用某个工具的结构化指令。
type ToolRequest struct {
	Name  string
	Input json.RawMessage
}

// ModelAdapter 定义 Runtime 依赖的统一模型适配接口。
type ModelAdapter interface {
	Run(ctx context.Context, input ModelInput) (ModelOutput, error)
}
