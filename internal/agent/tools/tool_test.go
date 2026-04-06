package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestToolContractSupportsStubImplementation(t *testing.T) {
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	input := CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"amount":10}`),
		Now:       now,
	}

	var tool Tool = stubTool{}

	spec := tool.Spec()
	if spec.Name != "stub_tool" {
		t.Fatalf("expected tool name %q, got %q", "stub_tool", spec.Name)
	}
	if spec.Description == "" {
		t.Fatal("expected tool description to be populated")
	}
	if spec.InputSchema["type"] != "object" {
		t.Fatalf("expected input schema type object, got %v", spec.InputSchema["type"])
	}

	output, err := tool.Call(context.Background(), input)
	if err != nil {
		t.Fatalf("expected tool call to succeed, got %v", err)
	}
	if output.ToolName != "stub_tool" {
		t.Fatalf("expected output tool name %q, got %q", "stub_tool", output.ToolName)
	}

	result, ok := output.Content.(map[string]any)
	if !ok {
		t.Fatalf("expected output content to be a map, got %T", output.Content)
	}
	if result["session_id"] != "session-1" {
		t.Fatalf("expected session_id %q, got %v", "session-1", result["session_id"])
	}
	if result["called_at"] != now.Format(time.RFC3339Nano) {
		t.Fatalf("expected called_at %q, got %v", now.Format(time.RFC3339Nano), result["called_at"])
	}
	if result["raw"] != `{"amount":10}` {
		t.Fatalf("expected raw payload to be preserved, got %v", result["raw"])
	}
}

func TestToolErrorsAreExposed(t *testing.T) {
	if !errors.Is(ErrToolNotFound, ErrToolNotFound) {
		t.Fatal("expected ErrToolNotFound to be defined")
	}
	if !errors.Is(ErrInvalidToolInput, ErrInvalidToolInput) {
		t.Fatal("expected ErrInvalidToolInput to be defined")
	}
	if !errors.Is(ErrDuplicateTool, ErrDuplicateTool) {
		t.Fatal("expected ErrDuplicateTool to be defined")
	}
}

type stubTool struct{}

func (stubTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "stub_tool",
		Description: "用于验证工具协议层的最小桩工具",
		InputSchema: map[string]any{
			"type": "object",
		},
	}
}

func (stubTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx

	return CallOutput{
		ToolName: "stub_tool",
		Content: map[string]any{
			"session_id": input.SessionID,
			"called_at":  input.Now.Format(time.RFC3339Nano),
			"raw":        string(input.Raw),
		},
	}, nil
}
