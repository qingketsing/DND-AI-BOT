package runtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/agent/tools"
	"DND-AI-BOT/internal/observability"
)

const (
	defaultMaxSteps         = 8
	defaultContextMax       = 40
	defaultToolFailureLimit = 2
	runtimeStepLogThreshold = 10 * time.Second
)

// Runtime 负责串联模型调用、工具执行和步骤积累。
type Runtime struct {
	model              ModelAdapter
	registry           tools.Registry
	executor           tools.Executor
	metrics            *observability.Metrics
	logger             *slog.Logger
	modelCallLogConfig RuntimeModelCallLogConfig
}

type RuntimeOption func(*Runtime)

func WithRuntimeMetrics(metrics *observability.Metrics) RuntimeOption {
	return func(runtime *Runtime) {
		if metrics != nil {
			runtime.metrics = metrics
		}
	}
}

func WithRuntimeLogger(logger *slog.Logger) RuntimeOption {
	return func(runtime *Runtime) {
		if logger != nil {
			runtime.logger = logger
		}
	}
}

func WithRuntimeModelCallLogConfig(config RuntimeModelCallLogConfig) RuntimeOption {
	return func(runtime *Runtime) {
		runtime.modelCallLogConfig = normalizeRuntimeModelCallLogConfig(config)
	}
}

type runtimeStepBreakdown struct {
	StepIndex  int
	ToolName   string
	OutputType string
	Status     string
	ModelTime  time.Duration
	ToolTime   time.Duration
	TotalTime  time.Duration
}

type runtimeModelCallEvent struct {
	StepIndex  int
	Status     string
	OutputType string
	ToolName   string
	Duration   time.Duration
	Err        error
}

// NewRuntime 创建一个支持 ReAct 循环的 Runtime。
func NewRuntime(model ModelAdapter, registry tools.Registry, executor tools.Executor, options ...RuntimeOption) *Runtime {
	runtime := &Runtime{
		model:              model,
		registry:           registry,
		executor:           executor,
		modelCallLogConfig: DefaultRuntimeModelCallLogConfig(),
	}
	for _, option := range options {
		if option != nil {
			option(runtime)
		}
	}
	return runtime
}

// Run 执行一轮 ReAct 流程，直到拿到最终回复或超出步数上限。
func (r *Runtime) Run(ctx context.Context, input RuntimeInput) (RuntimeOutput, error) {
	if err := validateRuntimeInput(input); err != nil {
		return RuntimeOutput{}, err
	}
	input = normalizeRuntimeInput(input)

	toolSpecs := r.registry.List()
	steps := make([]StepRecord, 0, input.MaxSteps)
	toolFailureCount := 0
	modelCallCount := 0
	toolStepCount := 0

	for i := 0; i < input.MaxSteps; i++ {
		stepStartedAt := time.Now()
		modelInput := buildModelInput(input, toolSpecs, steps)
		modelStartedAt := time.Now()
		modelOutput, err := r.model.Run(ctx, modelInput)
		modelDuration := time.Since(modelStartedAt)
		modelCallCount++
		if err != nil {
			r.recordModelCall("error", "invalid", modelStartedAt)
			r.logModelCall(input, runtimeModelCallEvent{
				StepIndex:  i,
				Status:     "error",
				OutputType: "invalid",
				Duration:   modelDuration,
				Err:        err,
			})
			r.recordRuntimeStep("error", "invalid", stepStartedAt)
			r.recordRunCounts("error", modelCallCount, toolStepCount)
			r.logSlowRuntimeStep(input, runtimeStepBreakdown{
				StepIndex:  i,
				OutputType: "invalid",
				Status:     "error",
				ModelTime:  modelDuration,
				TotalTime:  time.Since(stepStartedAt),
			})
			return RuntimeOutput{}, err
		}
		if err := validateModelOutput(modelOutput); err != nil {
			r.recordModelCall("error", "invalid", modelStartedAt)
			r.logModelCall(input, runtimeModelCallEvent{
				StepIndex:  i,
				Status:     "error",
				OutputType: "invalid",
				Duration:   modelDuration,
				Err:        err,
			})
			r.recordRuntimeStep("error", "invalid", stepStartedAt)
			r.recordRunCounts("error", modelCallCount, toolStepCount)
			r.logSlowRuntimeStep(input, runtimeStepBreakdown{
				StepIndex:  i,
				OutputType: "invalid",
				Status:     "error",
				ModelTime:  modelDuration,
				TotalTime:  time.Since(stepStartedAt),
			})
			return RuntimeOutput{}, err
		}

		outputType := classifyModelOutput(modelOutput)
		r.recordModelCall("success", outputType, modelStartedAt)
		r.logModelCall(input, runtimeModelCallEvent{
			StepIndex:  i,
			Status:     "success",
			OutputType: outputType,
			ToolName:   toolNameFromModelOutput(modelOutput),
			Duration:   modelDuration,
		})
		if isTerminalModelOutput(modelOutput) {
			r.recordRuntimeStep("success", outputType, stepStartedAt)
			r.recordRunCounts("success", modelCallCount, toolStepCount)
			r.logSlowRuntimeStep(input, runtimeStepBreakdown{
				StepIndex:  i,
				OutputType: outputType,
				Status:     "success",
				ModelTime:  modelDuration,
				TotalTime:  time.Since(stepStartedAt),
			})
			return RuntimeOutput{
				Reply: modelOutput.Reply,
				Steps: steps,
			}, nil
		}

		toolStartedAt := time.Now()
		toolOutput, err := r.executor.Execute(ctx, modelOutput.ToolRequest.Name, tools.CallInput{
			SessionID: input.SessionID,
			Raw:       modelOutput.ToolRequest.Input,
			Now:       time.Now().UTC(),
		})
		toolDuration := time.Since(toolStartedAt)
		toolStepCount++
		if err != nil {
			r.recordToolStep(modelOutput.ToolRequest.Name, "error", toolStartedAt)
			r.recordRuntimeStep("error", outputType, stepStartedAt)
			r.logSlowRuntimeStep(input, runtimeStepBreakdown{
				StepIndex:  i,
				ToolName:   modelOutput.ToolRequest.Name,
				OutputType: outputType,
				Status:     "error",
				ModelTime:  modelDuration,
				ToolTime:   toolDuration,
				TotalTime:  time.Since(stepStartedAt),
			})
			toolFailureCount++
			steps = append(steps, buildToolErrorStepRecord(modelOutput, err))
			if toolFailureCount >= defaultToolFailureLimit {
				r.recordRunCounts("tool_failure_limit", modelCallCount, toolStepCount)
				return RuntimeOutput{}, ErrToolFailureLimitExceeded
			}
			continue
		}
		toolFailureCount = 0
		r.recordToolStep(modelOutput.ToolRequest.Name, "success", toolStartedAt)
		r.recordRuntimeStep("success", outputType, stepStartedAt)
		r.logSlowRuntimeStep(input, runtimeStepBreakdown{
			StepIndex:  i,
			ToolName:   modelOutput.ToolRequest.Name,
			OutputType: outputType,
			Status:     "success",
			ModelTime:  modelDuration,
			ToolTime:   toolDuration,
			TotalTime:  time.Since(stepStartedAt),
		})

		steps = append(steps, buildStepRecord(modelOutput, toolOutput))
	}

	r.recordRunCounts("step_limit", modelCallCount, toolStepCount)
	return RuntimeOutput{}, ErrStepLimitExceeded
}

// normalizeRuntimeInput 为 Runtime 输入补齐默认配置。
func normalizeRuntimeInput(input RuntimeInput) RuntimeInput {
	if input.MaxSteps <= 0 {
		input.MaxSteps = defaultMaxSteps
	}
	if input.ContextLimit <= 0 {
		input.ContextLimit = defaultContextMax
	}
	return input
}

// validateRuntimeInput 校验 Runtime 的最小必填输入。
func validateRuntimeInput(input RuntimeInput) error {
	if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.UserMessage) == "" {
		return ErrInvalidRuntimeInput
	}
	return nil
}

// buildModelInput 将当前 Runtime 状态转换为模型输入。
func buildModelInput(input RuntimeInput, toolSpecs []tools.ToolSpec, steps []StepRecord) ModelInput {
	copiedSteps := append([]StepRecord(nil), steps...)
	return ModelInput{
		SessionID:    input.SessionID,
		SystemPrompt: input.SystemPrompt,
		UserMessage:  input.UserMessage,
		Tools:        toolSpecs,
		Steps:        copiedSteps,
	}
}

// isTerminalModelOutput 判断模型输出是否已经给出最终回复。
func isTerminalModelOutput(output ModelOutput) bool {
	return strings.TrimSpace(output.Reply) != ""
}

// isToolRequestModelOutput 判断模型输出是否请求了工具调用。
func isToolRequestModelOutput(output ModelOutput) bool {
	return output.ToolRequest != nil
}

// validateModelOutput 校验模型输出是否符合单轮协议约束。
func validateModelOutput(output ModelOutput) error {
	hasReply := strings.TrimSpace(output.Reply) != ""
	hasToolRequest := isToolRequestModelOutput(output)

	if hasReply == hasToolRequest {
		return ErrInvalidModelOutput
	}
	if hasToolRequest && strings.TrimSpace(output.ToolRequest.Name) == "" {
		return ErrInvalidModelOutput
	}
	return nil
}

func classifyModelOutput(output ModelOutput) string {
	if isTerminalModelOutput(output) {
		return "final_reply"
	}
	if isToolRequestModelOutput(output) {
		return "tool_request"
	}
	return "invalid"
}

func toolNameFromModelOutput(output ModelOutput) string {
	if output.ToolRequest == nil {
		return ""
	}
	return output.ToolRequest.Name
}

func (r *Runtime) recordModelCall(status string, outputType string, startedAt time.Time) {
	if r.metrics == nil {
		return
	}
	observability.ObserveDuration(r.metrics.RuntimeModelCallDuration, prometheus.Labels{
		"status":      status,
		"output_type": outputType,
	}, startedAt)
}

func (r *Runtime) recordToolStep(toolName string, status string, startedAt time.Time) {
	if r.metrics == nil {
		return
	}
	observability.ObserveDuration(r.metrics.RuntimeToolStepDuration, prometheus.Labels{
		"tool":   toolName,
		"status": status,
	}, startedAt)
}

func (r *Runtime) recordRuntimeStep(status string, outputType string, startedAt time.Time) {
	if r.metrics == nil {
		return
	}
	observability.ObserveDuration(r.metrics.RuntimeStepDuration, prometheus.Labels{
		"status":      status,
		"output_type": outputType,
	}, startedAt)
}

func (r *Runtime) recordRunCounts(status string, modelCalls int, toolSteps int) {
	if r.metrics == nil {
		return
	}
	labels := prometheus.Labels{"status": status}
	observability.ObserveHistogram(r.metrics.RuntimeModelCallsPerRun, labels, float64(modelCalls))
	observability.ObserveHistogram(r.metrics.RuntimeToolStepsPerRun, labels, float64(toolSteps))
}

func (r *Runtime) logSlowRuntimeStep(input RuntimeInput, step runtimeStepBreakdown) {
	if r.logger == nil || step.TotalTime < runtimeStepLogThreshold {
		return
	}
	r.logger.Warn(
		"runtime step latency",
		"session_id", input.SessionID,
		"step_index", step.StepIndex,
		"tool", step.ToolName,
		"output_type", step.OutputType,
		"status", step.Status,
		"model_ms", step.ModelTime.Milliseconds(),
		"tool_ms", step.ToolTime.Milliseconds(),
		"total_ms", step.TotalTime.Milliseconds(),
	)
}

func (r *Runtime) logModelCall(input RuntimeInput, event runtimeModelCallEvent) {
	if r.logger == nil || !shouldLogRuntimeModelCall(r.modelCallLogConfig, event.Duration, event.Status) {
		return
	}
	attrs := []any{
		"session_id", input.SessionID,
		"step_index", event.StepIndex,
		"status", event.Status,
		"output_type", event.OutputType,
		"tool", event.ToolName,
		"duration_ms", event.Duration.Milliseconds(),
	}
	if event.Err != nil {
		attrs = append(attrs, "error", event.Err)
		r.logger.Warn("runtime model call failed", attrs...)
		return
	}
	r.logger.Info("runtime model call completed", attrs...)
}

func shouldLogRuntimeModelCall(config RuntimeModelCallLogConfig, duration time.Duration, status string) bool {
	config = normalizeRuntimeModelCallLogConfig(config)
	if status == "error" {
		return true
	}
	switch config.Mode {
	case RuntimeModelCallLogOff:
		return false
	case RuntimeModelCallLogAll:
		return true
	case RuntimeModelCallLogSlow:
		return duration >= config.Threshold
	default:
		return duration >= config.Threshold
	}
}

// buildStepRecord 将模型动作和工具观察结果合并为单个步骤记录。
func buildStepRecord(output ModelOutput, toolOutput tools.CallOutput) StepRecord {
	return StepRecord{
		Thought:          output.Thought,
		ReasoningContent: output.ReasoningContent,
		ActionName:       output.ToolRequest.Name,
		ActionInput:      output.ToolRequest.Input,
		Observation:      toolOutput.Content,
	}
}

func buildToolErrorStepRecord(output ModelOutput, err error) StepRecord {
	toolName := ""
	var input json.RawMessage
	if output.ToolRequest != nil {
		toolName = output.ToolRequest.Name
		input = output.ToolRequest.Input
	}
	return StepRecord{
		Thought:          output.Thought,
		ReasoningContent: output.ReasoningContent,
		ActionName:       toolName,
		ActionInput:      input,
		Observation: ToolErrorObservation{
			ToolName:  toolName,
			Message:   err.Error(),
			Retryable: true,
		},
	}
}
