package tools

import "context"

// Executor 定义统一的工具执行入口。
type Executor interface {
	Execute(ctx context.Context, toolName string, input CallInput) (CallOutput, error)
}

// DefaultExecutor 基于注册表按名称查找并执行工具。
type DefaultExecutor struct {
	registry Registry
}

// NewExecutor 创建一个默认工具执行器。
func NewExecutor(registry Registry) *DefaultExecutor {
	return &DefaultExecutor{registry: registry}
}

// Execute 查找指定工具并执行其调用逻辑。
func (e *DefaultExecutor) Execute(ctx context.Context, toolName string, input CallInput) (CallOutput, error) {
	tool, ok := e.registry.Get(toolName)
	if !ok {
		return CallOutput{}, ErrToolNotFound
	}

	return tool.Call(ctx, input)
}
