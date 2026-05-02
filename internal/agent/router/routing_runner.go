package router

import (
	"context"
	"log/slog"

	"DND-AI-BOT/internal/agent/client"
	"DND-AI-BOT/internal/agent/intent"
	"DND-AI-BOT/internal/service"
)

// RoleRunnerMap 保存不同模型角色对应的执行器。
type RoleRunnerMap map[client.ModelRole]service.AgentRunner

// NewRoutingAgentRunner 创建按用户意图选择模型角色的 AgentRunner。
func NewRoutingAgentRunner(
	classifier intent.Classifier,
	policy Policy,
	runners RoleRunnerMap,
	logger *slog.Logger,
) service.AgentRunner {
	runner := &RoutingAgentRunner{
		classifier: classifier,
		policy:     policy,
		runners:    runners,
		logger:     logger,
	}
	return runner.Run
}

// RoutingAgentRunner 将分类、策略和实际 runner 解耦。
type RoutingAgentRunner struct {
	classifier intent.Classifier
	policy     Policy
	runners    RoleRunnerMap
	logger     *slog.Logger
}

// Run 根据当前用户输入选择 fast 或 primary runner。
func (r *RoutingAgentRunner) Run(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
	classifier := r.classifier
	if classifier == nil {
		classifier = intent.NewKeywordClassifier()
	}
	result := classifier.Classify(input.UserMessage)
	decision := r.policy.Decide(result)

	runner, selectedRole := r.selectRunner(decision.ModelRole)
	routedInput := input
	routedInput.MaxSteps = decision.MaxSteps

	if r.logger != nil {
		r.logger.Info(
			"agent route selected",
			"session_id", input.SessionID,
			"intent_kind", string(result.Kind),
			"intent_confidence", result.Confidence,
			"model_role", string(selectedRole),
			"requested_model_role", string(decision.ModelRole),
			"max_steps", routedInput.MaxSteps,
			"reason", string(decision.Reason),
		)
	}

	return runner(ctx, routedInput)
}

func (r *RoutingAgentRunner) selectRunner(role client.ModelRole) (service.AgentRunner, client.ModelRole) {
	if r.runners != nil {
		if runner := r.runners[role]; runner != nil {
			return runner, role
		}
		if runner := r.runners[client.ModelRolePrimary]; runner != nil {
			return runner, client.ModelRolePrimary
		}
	}
	return func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		return service.AgentReplyResult{}, service.ErrInvalidAgentService
	}, client.ModelRolePrimary
}
