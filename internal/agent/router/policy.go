package router

import (
	"DND-AI-BOT/internal/agent/client"
	"DND-AI-BOT/internal/agent/intent"
)

// RouteReason 表示路由决策原因。
type RouteReason string

// Decision 表示一次模型路由决策。
type Decision struct {
	ModelRole client.ModelRole
	MaxSteps  int
	Reason    RouteReason
}

// Policy 定义从意图到模型角色的路由策略。
type Policy struct {
	FastMaxSteps    int
	PrimaryMaxSteps int
}

// DefaultPolicy 返回生产默认路由策略。
func DefaultPolicy() Policy {
	return Policy{FastMaxSteps: 2, PrimaryMaxSteps: 8}
}

// Decide 根据意图分类结果选择模型角色。
func (p Policy) Decide(result intent.Result) Decision {
	p = normalizePolicy(p)
	switch result.Kind {
	case intent.KindStatusQuery, intent.KindSessionRecall, intent.KindCharacterDraft:
		return Decision{ModelRole: client.ModelRoleFast, MaxSteps: p.FastMaxSteps, Reason: "lightweight_intent"}
	default:
		return Decision{ModelRole: client.ModelRolePrimary, MaxSteps: p.PrimaryMaxSteps, Reason: "primary_required"}
	}
}

func normalizePolicy(policy Policy) Policy {
	if policy.FastMaxSteps <= 0 {
		policy.FastMaxSteps = 2
	}
	if policy.PrimaryMaxSteps <= 0 {
		policy.PrimaryMaxSteps = 8
	}
	return policy
}
