package service

import (
	"strings"

	agentintent "DND-AI-BOT/internal/agent/intent"
)

// AgentFailureKind 标识 Agent 运行失败发生在哪一类能力上。
type AgentFailureKind string

const (
	AgentFailureModel   AgentFailureKind = "model"
	AgentFailureRAG     AgentFailureKind = "rag"
	AgentFailureTool    AgentFailureKind = "tool"
	AgentFailureRuntime AgentFailureKind = "runtime"
)

// AgentFallbackInput 定义构造兜底回复所需的最小上下文。
type AgentFallbackInput struct {
	SessionID   string
	UserMessage string
	Err         error
	Kind        AgentFailureKind
}

// AgentFallbackReply 表示兜底回复结果。
type AgentFallbackReply struct {
	Reply string
	Kind  AgentFailureKind
}

// AgentFallbackResponder 定义 Agent 失败后的兜底回复能力。
type AgentFallbackResponder interface {
	BuildFallbackReply(input AgentFallbackInput) AgentFallbackReply
}

// DefaultAgentFallbackResponder 返回面向玩家可展示的保守兜底回复。
type DefaultAgentFallbackResponder struct{}

// NewDefaultAgentFallbackResponder 创建默认 Agent 兜底回复器。
func NewDefaultAgentFallbackResponder() *DefaultAgentFallbackResponder {
	return &DefaultAgentFallbackResponder{}
}

// BuildFallbackReply 根据失败类型构造兜底回复。
func (r *DefaultAgentFallbackResponder) BuildFallbackReply(input AgentFallbackInput) AgentFallbackReply {
	kind := input.Kind
	if kind == "" {
		kind = inferAgentFailureKind(input.Err)
	}
	intentResult := agentintent.NewKeywordClassifier().Classify(input.UserMessage)

	reply := "当前模型服务暂时不可用，我已经保存了你的消息。请稍后重试；如果你刚才尝试更新角色、战斗或任务状态，该状态可能尚未写入成功。"
	switch kind {
	case AgentFailureRAG:
		reply = "知识库检索暂时不可用。我会先基于当前会话上下文和已确认游戏状态继续；涉及具体规则或设定细节时，需要稍后重新检索确认。"
	case AgentFailureTool:
		reply = buildToolFailureReply(intentResult.Kind)
	case AgentFailureRuntime:
		reply = buildRuntimeFailureReply(intentResult.Kind)
	}

	return AgentFallbackReply{
		Reply: reply,
		Kind:  kind,
	}
}

func buildToolFailureReply(kind agentintent.Kind) string {
	switch kind {
	case agentintent.KindCombatAction:
		return "我尝试结算这次战斗动作时失败了，所以本次攻击或行动尚未结算，当前回合也没有推进。请直接重试刚才的动作，我会以未推进的状态继续。"
	case agentintent.KindExplorationAction:
		return "我尝试写入这次场景动作时失败了，所以你刚才的动作尚未正式落地，场景也没有继续推进。请直接重试，我会从当前未推进的位置继续。"
	case agentintent.KindStatusQuery, agentintent.KindSessionRecall:
		return "我刚才无法可靠读取当前的结构化状态，所以不会编造血量、位置或回合信息。请稍后重试，或让我重新读取当前状态后再继续回答。"
	default:
		return "我尝试更新结构化游戏状态时失败了，因此不会假装状态已经改变。请稍后重试，或让我重新读取当前状态后继续。"
	}
}

func buildRuntimeFailureReply(kind agentintent.Kind) string {
	switch kind {
	case agentintent.KindCombatAction:
		return "当前 Agent 在处理这次战斗动作时暂时失败，所以本次攻击或行动还没有结算，回合也没有推进。请直接重试刚才的动作，我会按未推进状态继续。"
	case agentintent.KindExplorationAction:
		return "当前 Agent 在处理这次场景动作时暂时失败，所以门、移动或调查结果都还没有正式推进。请直接重试刚才的动作，我会从当前状态继续。"
	case agentintent.KindStatusQuery, agentintent.KindSessionRecall:
		return "当前 Agent 暂时无法可靠读取这一轮需要的结构化状态，所以我现在不能确认血量、位置、任务或刚才的推进结果。请稍后重试，或让我重新读取当前状态后继续。"
	default:
		return "当前 Agent 运行暂时失败，我已经保存了你的消息。请稍后重试；如果刚才涉及状态更新，该状态可能尚未写入成功。"
	}
}

func inferAgentFailureKind(err error) AgentFailureKind {
	if err == nil {
		return AgentFailureRuntime
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "search"), strings.Contains(message, "embedding"), strings.Contains(message, "rag"):
		return AgentFailureRAG
	case strings.Contains(message, "tool"), strings.Contains(message, "encounter not found"):
		return AgentFailureTool
	case strings.Contains(message, "chat completion"), strings.Contains(message, "model"), strings.Contains(message, "deepseek"), strings.Contains(message, "openai"):
		return AgentFailureModel
	default:
		return AgentFailureRuntime
	}
}
