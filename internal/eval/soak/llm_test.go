package soak

import (
	"context"
	"errors"
	"strings"
	"testing"

	agentclient "DND-AI-BOT/internal/agent/client"
	agentmock "DND-AI-BOT/internal/agent/client/mock"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
)

func TestPlayerSimulatorNextInputUsesModelReply(t *testing.T) {
	adapter := agentmock.NewAdapter([]agentruntime.ModelOutput{{Reply: "检查房间里的门"}})
	simulator := NewPlayerSimulator(adapter)

	output, err := simulator.NextInput(context.Background(), PlayerInput{
		SessionID:     "session-1",
		Scenario:      ScenarioConfig{Name: "the-city", Objective: "探索"},
		Round:         1,
		Records:       []RoundRecord{{Round: 1, AgentReply: "你看到一扇门"}},
		GameObjective: "探索",
	})
	if err != nil {
		t.Fatalf("expected player simulator to succeed, got %v", err)
	}

	if output != "检查房间里的门" {
		t.Fatalf("expected model reply, got %q", output)
	}
	if len(adapter.Inputs()) != 1 {
		t.Fatalf("expected one model call, got %d", len(adapter.Inputs()))
	}
	if adapter.Inputs()[0].SessionID != "session-1" {
		t.Fatalf("expected session id to be forwarded, got %q", adapter.Inputs()[0].SessionID)
	}
	if !strings.Contains(adapter.Inputs()[0].UserMessage, "你看到一扇门") {
		t.Fatalf("expected previous turn context in prompt, got %q", adapter.Inputs()[0].UserMessage)
	}
}

func TestJudgeEvaluateParsesJSONReply(t *testing.T) {
	adapter := agentmock.NewAdapter([]agentruntime.ModelOutput{{Reply: `{"success":true,"score":0.95,"failure_reasons":[],"comment":"ok"}`}})
	judge := NewJudge(adapter)

	result, err := judge.Evaluate(context.Background(), JudgeInput{
		Scenario:   ScenarioConfig{Name: "the-city"},
		Round:      1,
		UserInput:  "继续探索",
		AgentReply: "你继续前进",
	})
	if err != nil {
		t.Fatalf("expected judge to succeed, got %v", err)
	}

	if !result.Success {
		t.Fatal("expected success")
	}
	if result.Score != 0.95 {
		t.Fatalf("expected score 0.95, got %f", result.Score)
	}
}

func TestJudgeEvaluateReturnsErrorWhenReplyIsNotJSON(t *testing.T) {
	adapter := agentmock.NewAdapter([]agentruntime.ModelOutput{{Reply: "not json"}})
	judge := NewJudge(adapter)

	_, err := judge.Evaluate(context.Background(), JudgeInput{
		SessionID:  "session-1",
		Scenario:   ScenarioConfig{Name: "the-city"},
		Round:      1,
		UserInput:  "继续探索",
		AgentReply: "你继续前进",
	})
	if err == nil {
		t.Fatal("expected invalid judge JSON to return an error")
	}
	if !strings.Contains(err.Error(), "parse judge JSON") {
		t.Fatalf("expected parse error context, got %v", err)
	}
}

func TestBuildModelAdapterSupportsMockProvider(t *testing.T) {
	adapter, err := BuildModelAdapter(ModelConfig{Provider: string(agentclient.ProviderMock)})
	if err != nil {
		t.Fatalf("expected mock adapter build to succeed, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected adapter")
	}
}

func TestBuildModelAdapterRejectsUnsupportedProvider(t *testing.T) {
	_, err := BuildModelAdapter(ModelConfig{Provider: "unknown"})
	if !errors.Is(err, agentclient.ErrUnsupportedProvider) {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}
