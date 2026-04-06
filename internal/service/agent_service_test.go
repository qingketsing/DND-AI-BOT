package service

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
)

func TestAgentServiceReplyReturnsRuntimeOutput(t *testing.T) {
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{
			Reply: "你当前背包里有一瓶治疗药水。",
			Steps: []AgentStep{{ToolName: "get_game_state"}},
		}, nil
	}, nil)

	output, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "我的背包里有什么？",
		MaxSteps:    4,
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}
	if output.Reply != "你当前背包里有一瓶治疗药水。" {
		t.Fatalf("expected final reply, got %q", output.Reply)
	}
	if len(output.Steps) != 1 {
		t.Fatalf("expected one tool step, got %+v", output.Steps)
	}
}

func TestAgentServiceReplyLogsSuccess(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{
			Reply: "你当前背包里有一瓶治疗药水。",
			Steps: []AgentStep{{ToolName: "get_game_state"}},
		}, nil
	}, logger)

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "我的背包里有什么？",
		MaxSteps:    4,
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "agent run started") {
		t.Fatalf("expected start log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "agent run finished") {
		t.Fatalf("expected finish log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "get_game_state") {
		t.Fatalf("expected tool name to appear in logs, got %q", logOutput)
	}
}

func TestAgentServiceReplyLogsFailure(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{}, errors.New("boom")
	}, logger)

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err == nil {
		t.Fatal("expected reply to fail")
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "agent run failed") {
		t.Fatalf("expected failure log, got %q", logOutput)
	}
}

func TestToolNamesFromStepsReturnsCalledTools(t *testing.T) {
	names := toolNamesFromSteps([]AgentStep{
		{ToolName: "get_game_state"},
		{ToolName: ""},
		{ToolName: "skill_check"},
	})

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}
	if names[0] != "get_game_state" || names[1] != "skill_check" {
		t.Fatalf("unexpected tool names %v", names)
	}
}
