package service

import (
	"context"
	"errors"
	"log"
	"strings"
)

var (
	// ErrInvalidAgentService 表示 AgentService 缺少可用的 Runtime 依赖。
	ErrInvalidAgentService = errors.New("invalid agent service")
	// ErrInvalidAgentReplyInput 表示发起 Agent 回复时传入了不合法参数。
	ErrInvalidAgentReplyInput = errors.New("invalid agent reply input")
)

const (
	defaultLoggedMaxSteps     = 8
	defaultLoggedContextLimit = 40
)

// AgentService 负责调 Runtime 完成一轮 Agent 回复，并集中记录运行日志。
type AgentService struct {
	runner AgentRunner
	logger *log.Logger
}

// AgentReplyInput 定义发起一轮 Agent 回复所需的最小输入。
type AgentReplyInput struct {
	SessionID    string
	SystemPrompt string
	UserMessage  string
	MaxSteps     int
	ContextLimit int
}

// AgentStep 表示一轮 Agent 回复中产生的单个工具步骤摘要。
type AgentStep struct {
	ToolName string
}

// AgentReplyResult 定义 AgentService 返回给业务层的最小结果结构。
type AgentReplyResult struct {
	Reply string
	Steps []AgentStep
}

// AgentRunner 抽象一轮 Agent 执行能力，便于由 app 层适配 Runtime。
type AgentRunner func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error)

// NewAgentService 创建一个可复用的 Agent 服务。
func NewAgentService(runner AgentRunner, logger *log.Logger) *AgentService {
	return &AgentService{
		runner: runner,
		logger: logger,
	}
}

// Reply 执行一轮 Agent 运行，并返回 Runtime 的最终输出。
func (s *AgentService) Reply(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
	if s.runner == nil {
		return AgentReplyResult{}, ErrInvalidAgentService
	}
	if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.UserMessage) == "" {
		return AgentReplyResult{}, ErrInvalidAgentReplyInput
	}

	effectiveInput := normalizeAgentReplyInputForLogging(input)
	s.logRunStarted(effectiveInput)

	output, err := s.runner(ctx, input)
	if err != nil {
		s.logRunFailed(effectiveInput, err)
		return AgentReplyResult{}, err
	}

	s.logRunFinished(effectiveInput, output)
	return output, nil
}

// logRunStarted 记录一轮 Agent 执行开始时的最小上下文。
func (s *AgentService) logRunStarted(input AgentReplyInput) {
	if s.logger == nil {
		return
	}

	s.logger.Printf(
		"agent run started: session_id=%s max_steps=%d context_limit=%d",
		input.SessionID,
		input.MaxSteps,
		input.ContextLimit,
	)
}

// logRunFinished 记录一轮 Agent 成功结束时的执行摘要。
func (s *AgentService) logRunFinished(input AgentReplyInput, output AgentReplyResult) {
	if s.logger == nil {
		return
	}

	s.logger.Printf(
		"agent run finished: session_id=%s step_count=%d tools=%v reply_length=%d",
		input.SessionID,
		len(output.Steps),
		toolNamesFromSteps(output.Steps),
		len(output.Reply),
	)
}

// logRunFailed 记录一轮 Agent 执行失败时的错误摘要。
func (s *AgentService) logRunFailed(input AgentReplyInput, err error) {
	if s.logger == nil {
		return
	}

	s.logger.Printf("agent run failed: session_id=%s error=%v", input.SessionID, err)
}

// toolNamesFromSteps 从步骤记录中提取出本轮真正调用过的工具名。
func toolNamesFromSteps(steps []AgentStep) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		if strings.TrimSpace(step.ToolName) == "" {
			continue
		}
		names = append(names, step.ToolName)
	}
	return names
}

func normalizeAgentReplyInputForLogging(input AgentReplyInput) AgentReplyInput {
	if input.MaxSteps <= 0 {
		input.MaxSteps = defaultLoggedMaxSteps
	}
	if input.ContextLimit <= 0 {
		input.ContextLimit = defaultLoggedContextLimit
	}
	return input
}
