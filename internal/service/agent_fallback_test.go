package service

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultAgentFallbackResponderCombatToolFailurePreservesTurnState(t *testing.T) {
	responder := NewDefaultAgentFallbackResponder()

	result := responder.BuildFallbackReply(AgentFallbackInput{
		SessionID:   "session-1",
		UserMessage: "继续攻击，终结它",
		Err:         errors.New("tool failure limit exceeded"),
		Kind:        AgentFailureTool,
	})

	if result.Kind != AgentFailureTool {
		t.Fatalf("expected kind %q, got %q", AgentFailureTool, result.Kind)
	}
	for _, expected := range []string{"本次攻击", "尚未结算", "回合", "未推进"} {
		if !strings.Contains(result.Reply, expected) {
			t.Fatalf("expected combat fallback reply to contain %q, got %q", expected, result.Reply)
		}
	}
}

func TestDefaultAgentFallbackResponderStatusQueryFailureDoesNotInventState(t *testing.T) {
	responder := NewDefaultAgentFallbackResponder()

	result := responder.BuildFallbackReply(AgentFallbackInput{
		SessionID:   "session-1",
		UserMessage: "它现在还有多少血量？",
		Err:         errors.New("runtime failed"),
		Kind:        AgentFailureRuntime,
	})

	for _, expected := range []string{"无法可靠读取", "结构化状态"} {
		if !strings.Contains(result.Reply, expected) {
			t.Fatalf("expected status-query fallback reply to contain %q, got %q", expected, result.Reply)
		}
	}
}

func TestDefaultAgentFallbackResponderExplorationToolFailureDoesNotAdvanceScene(t *testing.T) {
	responder := NewDefaultAgentFallbackResponder()

	result := responder.BuildFallbackReply(AgentFallbackInput{
		SessionID:   "session-1",
		UserMessage: "推开门继续前进",
		Err:         errors.New("tool update failed"),
		Kind:        AgentFailureTool,
	})

	for _, expected := range []string{"动作", "尚未", "场景", "推进"} {
		if !strings.Contains(result.Reply, expected) {
			t.Fatalf("expected exploration fallback reply to contain %q, got %q", expected, result.Reply)
		}
	}
}
