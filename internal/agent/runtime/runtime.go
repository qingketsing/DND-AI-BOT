package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"DND-AI-BOT/internal/agent/tools"
)

const (
	defaultMaxSteps         = 8
	defaultContextMax       = 40
	defaultToolFailureLimit = 2
)

// Runtime 负责串联模型调用、工具执行和步骤积累。
type Runtime struct {
	model    ModelAdapter
	registry tools.Registry
	executor tools.Executor
}

// NewRuntime 创建一个支持 ReAct 循环的 Runtime。
func NewRuntime(model ModelAdapter, registry tools.Registry, executor tools.Executor) *Runtime {
	return &Runtime{
		model:    model,
		registry: registry,
		executor: executor,
	}
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

	for i := 0; i < input.MaxSteps; i++ {
		modelInput := buildModelInput(input, toolSpecs, steps)
		modelOutput, err := r.model.Run(ctx, modelInput)
		if err != nil {
			return RuntimeOutput{}, err
		}
		if err := validateModelOutput(modelOutput); err != nil {
			return RuntimeOutput{}, err
		}

		if isTerminalModelOutput(modelOutput) {
			return RuntimeOutput{
				Reply: modelOutput.Reply,
				Steps: steps,
			}, nil
		}

		toolOutput, err := r.executor.Execute(ctx, modelOutput.ToolRequest.Name, tools.CallInput{
			SessionID: input.SessionID,
			Raw:       modelOutput.ToolRequest.Input,
			Now:       time.Now().UTC(),
		})
		if err != nil {
			toolFailureCount++
			steps = append(steps, buildToolErrorStepRecord(modelOutput, err))
			if toolFailureCount >= defaultToolFailureLimit {
				return RuntimeOutput{}, ErrToolFailureLimitExceeded
			}
			continue
		}
		toolFailureCount = 0

		steps = append(steps, buildStepRecord(modelOutput, toolOutput))
	}

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
