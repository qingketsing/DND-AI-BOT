package router

import (
	"context"
	"testing"

	"DND-AI-BOT/internal/agent/client"
	"DND-AI-BOT/internal/agent/intent"
	"DND-AI-BOT/internal/service"
)

func TestRoutingAgentRunnerUsesFastRunnerForStatusQuery(t *testing.T) {
	var fastCalled bool
	runner := NewRoutingAgentRunner(
		intent.NewKeywordClassifier(),
		DefaultPolicy(),
		RoleRunnerMap{
			client.ModelRolePrimary: func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
				t.Fatal("primary runner should not be called")
				return service.AgentReplyResult{}, nil
			},
			client.ModelRoleFast: func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
				fastCalled = true
				if input.MaxSteps != 2 {
					t.Fatalf("expected fast max steps 2, got %d", input.MaxSteps)
				}
				return service.AgentReplyResult{Reply: "fast reply"}, nil
			},
		},
		nil,
	)

	output, err := runner(context.Background(), service.AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "它还有多少血量？",
	})
	if err != nil {
		t.Fatalf("expected routing runner to succeed, got %v", err)
	}
	if !fastCalled {
		t.Fatal("expected fast runner to be called")
	}
	if output.Reply != "fast reply" {
		t.Fatalf("expected fast reply, got %q", output.Reply)
	}
}

func TestRoutingAgentRunnerUsesPrimaryRunnerForCombatAction(t *testing.T) {
	var primaryCalled bool
	runner := NewRoutingAgentRunner(
		intent.NewKeywordClassifier(),
		DefaultPolicy(),
		RoleRunnerMap{
			client.ModelRolePrimary: func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
				primaryCalled = true
				if input.MaxSteps != 8 {
					t.Fatalf("expected primary max steps 8, got %d", input.MaxSteps)
				}
				return service.AgentReplyResult{Reply: "primary reply"}, nil
			},
			client.ModelRoleFast: func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
				t.Fatal("fast runner should not be called")
				return service.AgentReplyResult{}, nil
			},
		},
		nil,
	)

	output, err := runner(context.Background(), service.AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "继续攻击",
	})
	if err != nil {
		t.Fatalf("expected routing runner to succeed, got %v", err)
	}
	if !primaryCalled {
		t.Fatal("expected primary runner to be called")
	}
	if output.Reply != "primary reply" {
		t.Fatalf("expected primary reply, got %q", output.Reply)
	}
}

func TestRoutingAgentRunnerFallsBackToPrimaryWhenFastRunnerMissing(t *testing.T) {
	var primaryCalled bool
	runner := NewRoutingAgentRunner(
		intent.NewKeywordClassifier(),
		DefaultPolicy(),
		RoleRunnerMap{
			client.ModelRolePrimary: func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
				primaryCalled = true
				if input.MaxSteps != 2 {
					t.Fatalf("expected fast decision max steps to be preserved, got %d", input.MaxSteps)
				}
				return service.AgentReplyResult{Reply: "primary fallback"}, nil
			},
		},
		nil,
	)

	output, err := runner(context.Background(), service.AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "它还有多少血量？",
	})
	if err != nil {
		t.Fatalf("expected routing runner to succeed, got %v", err)
	}
	if !primaryCalled {
		t.Fatal("expected primary runner to be called")
	}
	if output.Reply != "primary fallback" {
		t.Fatalf("expected primary fallback reply, got %q", output.Reply)
	}
}
