package tools

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

type ToolCallLogMode string

const (
	ToolCallLogOff  ToolCallLogMode = "off"
	ToolCallLogSlow ToolCallLogMode = "slow"
	ToolCallLogAll  ToolCallLogMode = "all"
)

type ToolCallLogConfig struct {
	Mode      ToolCallLogMode
	Threshold time.Duration
}

func DefaultToolCallLogConfig() ToolCallLogConfig {
	return ToolCallLogConfig{
		Mode:      ToolCallLogSlow,
		Threshold: time.Second,
	}
}

// Executor 定义统一的工具执行入口。
type Executor interface {
	Execute(ctx context.Context, toolName string, input CallInput) (CallOutput, error)
}

// DefaultExecutor 基于注册表按名称查找并执行工具。
type DefaultExecutor struct {
	registry          Registry
	metrics           *observability.Metrics
	logger            *slog.Logger
	toolCallLogConfig ToolCallLogConfig
}

// ExecutorOption 定义工具执行器可选配置。
type ExecutorOption func(*DefaultExecutor)

// NewExecutor 创建一个默认工具执行器。
func NewExecutor(registry Registry, options ...ExecutorOption) *DefaultExecutor {
	executor := &DefaultExecutor{
		registry:          registry,
		toolCallLogConfig: DefaultToolCallLogConfig(),
	}
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

// WithExecutorLogger 注入工具执行日志。
func WithExecutorLogger(logger *slog.Logger) ExecutorOption {
	return func(executor *DefaultExecutor) {
		if logger != nil {
			executor.logger = logger
		}
	}
}

func WithExecutorToolCallLogConfig(config ToolCallLogConfig) ExecutorOption {
	return func(executor *DefaultExecutor) {
		executor.toolCallLogConfig = normalizeToolCallLogConfig(config)
	}
}

func normalizeToolCallLogConfig(config ToolCallLogConfig) ToolCallLogConfig {
	if config.Mode != ToolCallLogOff && config.Mode != ToolCallLogSlow && config.Mode != ToolCallLogAll {
		config.Mode = ToolCallLogSlow
	}
	if config.Threshold < 0 {
		config.Threshold = time.Second
	}
	return config
}

// Execute 查找指定工具并执行其调用逻辑。
func (e *DefaultExecutor) Execute(ctx context.Context, toolName string, input CallInput) (CallOutput, error) {
	startedAt := time.Now()
	tool, ok := e.registry.Get(toolName)
	if !ok {
		e.recordToolCall(toolName, "error", startedAt)
		e.logToolFailure(toolName, input, ErrToolNotFound, startedAt)
		return CallOutput{}, ErrToolNotFound
	}

	output, err := tool.Call(ctx, input)
	if err != nil {
		e.recordToolCall(toolName, "error", startedAt)
		e.logToolFailure(toolName, input, err, startedAt)
		return CallOutput{}, err
	}
	e.recordToolCall(toolName, "success", startedAt)
	e.logToolSuccess(toolName, input, startedAt)
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

func (e *DefaultExecutor) logToolFailure(toolName string, input CallInput, err error, startedAt time.Time) {
	if e.logger == nil {
		return
	}
	e.logger.Warn(
		"tool execution failed",
		"tool", toolName,
		"session_id", input.SessionID,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"error", err,
	)
}

func (e *DefaultExecutor) logToolSuccess(toolName string, input CallInput, startedAt time.Time) {
	duration := time.Since(startedAt)
	if e.logger == nil || !e.shouldLogToolCall("success", duration) {
		return
	}
	e.logger.Info(
		"tool execution completed",
		"tool", toolName,
		"session_id", input.SessionID,
		"status", "success",
		"duration_ms", duration.Milliseconds(),
	)
}

func (e *DefaultExecutor) shouldLogToolCall(status string, duration time.Duration) bool {
	config := normalizeToolCallLogConfig(e.toolCallLogConfig)
	if status == "error" {
		return true
	}
	switch config.Mode {
	case ToolCallLogOff:
		return false
	case ToolCallLogAll:
		return true
	case ToolCallLogSlow:
		return duration >= config.Threshold
	default:
		return duration >= config.Threshold
	}
}
