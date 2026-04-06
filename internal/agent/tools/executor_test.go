package tools

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultExecutorExecuteRunsTargetTool(t *testing.T) {
	registry := NewInMemoryRegistry()
	tool := executorStubTool{
		name: "get_agent_context",
		output: CallOutput{
			ToolName: "get_agent_context",
			Content: map[string]any{
				"session_id": "session-1",
			},
		},
	}
	if err := registry.Register(tool); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry)
	output, err := executor.Execute(context.Background(), "get_agent_context", CallInput{
		SessionID: "session-1",
		Now:       time.Date(2026, 4, 6, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected execute to succeed, got %v", err)
	}
	if output.ToolName != "get_agent_context" {
		t.Fatalf("expected tool name %q, got %q", "get_agent_context", output.ToolName)
	}
	if tool.calledCount != 0 {
		t.Fatal("expected original stub copy not to be mutated")
	}
}

func TestDefaultExecutorExecuteReturnsToolNotFound(t *testing.T) {
	executor := NewExecutor(NewInMemoryRegistry())

	_, err := executor.Execute(context.Background(), "missing_tool", CallInput{})
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}

func TestDefaultExecutorExecutePropagatesToolError(t *testing.T) {
	registry := NewInMemoryRegistry()
	expectedErr := errors.New("boom")
	if err := registry.Register(executorStubTool{
		name: "spend_gold",
		err:  expectedErr,
	}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry)
	_, err := executor.Execute(context.Background(), "spend_gold", CallInput{
		SessionID: "session-1",
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected propagated tool error, got %v", err)
	}
}

type executorStubTool struct {
	name        string
	output      CallOutput
	err         error
	calledCount int
}

func (t executorStubTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: "执行器测试桩工具",
		InputSchema: map[string]any{
			"type": "object",
		},
	}
}

func (t executorStubTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx
	_ = input
	if t.err != nil {
		return CallOutput{}, t.err
	}
	return t.output, nil
}
