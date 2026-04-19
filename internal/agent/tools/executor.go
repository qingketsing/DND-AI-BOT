package tools

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

// Executor 定义统一的工具执行入口。
type Executor interface {
	Execute(ctx context.Context, toolName string, input CallInput) (CallOutput, error)
}

// DefaultExecutor 基于注册表按名称查找并执行工具。
type DefaultExecutor struct {
	registry Registry
	metrics  *observability.Metrics
}

// ExecutorOption 定义工具执行器可选配置。
type ExecutorOption func(*DefaultExecutor)

// NewExecutor 创建一个默认工具执行器。
func NewExecutor(registry Registry, options ...ExecutorOption) *DefaultExecutor {
	executor := &DefaultExecutor{registry: registry}
	for _, option := range options {
		if option != nil {
			option(executor)
		}
	}
	return executor
}

// WithExecutorMetrics 注入工具执行指标。
func WithExecutorMetrics(metrics *observability.Metrics) ExecutorOption {
	return func(executor *DefaultExecutor) {
		if metrics != nil {
			executor.metrics = metrics
		}
	}
}

// Execute 查找指定工具并执行其调用逻辑。
func (e *DefaultExecutor) Execute(ctx context.Context, toolName string, input CallInput) (CallOutput, error) {
	startedAt := time.Now()
	tool, ok := e.registry.Get(toolName)
	if !ok {
		e.recordToolCall(toolName, "error", startedAt)
		return CallOutput{}, ErrToolNotFound
	}

	output, err := tool.Call(ctx, input)
	if err != nil {
		e.recordToolCall(toolName, "error", startedAt)
		return CallOutput{}, err
	}
	e.recordToolCall(toolName, "success", startedAt)
	return output, nil
}

func (e *DefaultExecutor) recordToolCall(toolName string, status string, startedAt time.Time) {
	if e.metrics == nil {
		return
	}
	labels := prometheus.Labels{
		"tool":   toolName,
		"status": status,
	}
	e.metrics.ToolCallsTotal.With(labels).Inc()
	observability.ObserveDuration(e.metrics.ToolCallDuration, labels, startedAt)
	if status == "error" {
		e.metrics.ToolErrorsTotal.With(prometheus.Labels{"tool": toolName}).Inc()
	}
}
