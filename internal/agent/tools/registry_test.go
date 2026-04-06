package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestInMemoryRegistryRegisterAndGet(t *testing.T) {
	registry := NewInMemoryRegistry()
	tool := registryStubTool{name: "get_game_state", description: "读取当前游戏进度"}

	if err := registry.Register(tool); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	got, ok := registry.Get("get_game_state")
	if !ok {
		t.Fatal("expected tool to be retrievable after registration")
	}
	if got.Spec().Name != "get_game_state" {
		t.Fatalf("expected tool name %q, got %q", "get_game_state", got.Spec().Name)
	}
}

func TestInMemoryRegistryRegisterRejectsDuplicateTool(t *testing.T) {
	registry := NewInMemoryRegistry()
	tool := registryStubTool{name: "apply_damage", description: "对目标造成伤害"}

	if err := registry.Register(tool); err != nil {
		t.Fatalf("expected first register to succeed, got %v", err)
	}

	err := registry.Register(tool)
	if !errors.Is(err, ErrDuplicateTool) {
		t.Fatalf("expected ErrDuplicateTool, got %v", err)
	}
}

func TestInMemoryRegistryListReturnsSortedSpecs(t *testing.T) {
	registry := NewInMemoryRegistry()
	if err := registry.Register(registryStubTool{name: "skill_check", description: "技能检定"}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}
	if err := registry.Register(registryStubTool{name: "add_item", description: "添加物品"}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	specs := registry.List()
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
	if specs[0].Name != "add_item" || specs[1].Name != "skill_check" {
		t.Fatalf("expected specs to be sorted by name, got %+v", specs)
	}
}

func TestInMemoryRegistryRegisterRejectsEmptyToolName(t *testing.T) {
	registry := NewInMemoryRegistry()

	err := registry.Register(registryStubTool{name: "", description: "缺少名称"})
	if !errors.Is(err, ErrInvalidToolInput) {
		t.Fatalf("expected ErrInvalidToolInput, got %v", err)
	}
}

type registryStubTool struct {
	name        string
	description string
}

func (t registryStubTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: t.description,
		InputSchema: map[string]any{
			"type": "object",
		},
	}
}

func (t registryStubTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx
	_ = input
	return CallOutput{
		ToolName: t.name,
		Content:  nil,
	}, nil
}

func TestSortedToolNamesReturnsStableOrder(t *testing.T) {
	names := sortedToolNames(map[string]Tool{
		"z_tool": registryStubTool{name: "z_tool"},
		"a_tool": registryStubTool{name: "a_tool"},
	})

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "a_tool" || names[1] != "z_tool" {
		t.Fatalf("expected sorted names, got %+v", names)
	}
}

func TestRegistryExecutorCompatibility(t *testing.T) {
	registry := NewInMemoryRegistry()
	tool := registryStubTool{name: "roll_dice", description: "掷骰"}
	if err := registry.Register(tool); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry)
	output, err := executor.Execute(context.Background(), "roll_dice", CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"expression":"1d20"}`),
		Now:       time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected execute to succeed, got %v", err)
	}
	if output.ToolName != "roll_dice" {
		t.Fatalf("expected output tool name %q, got %q", "roll_dice", output.ToolName)
	}
}
