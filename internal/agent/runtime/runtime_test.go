package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"DND-AI-BOT/internal/agent/tools"
)

func TestRuntimeRunReturnsDirectReply(t *testing.T) {
	model := &fakeModelAdapter{
		outputs: []ModelOutput{
			{Reply: "你好，冒险者。"},
		},
	}
	registry := &fakeRegistry{
		specs: []tools.ToolSpec{
			{Name: "get_game_state", Description: "读取游戏进度"},
		},
	}
	executor := &fakeExecutor{}

	runtime := NewRuntime(model, registry, executor)
	output, err := runtime.Run(context.Background(), RuntimeInput{
		SessionID:   "session-1",
		UserMessage: "你好",
	})
	if err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}
	if output.Reply != "你好，冒险者。" {
		t.Fatalf("expected reply %q, got %q", "你好，冒险者。", output.Reply)
	}
	if len(output.Steps) != 0 {
		t.Fatalf("expected no steps for direct reply, got %+v", output.Steps)
	}
	if len(model.inputs) != 1 {
		t.Fatalf("expected model to be called once, got %d", len(model.inputs))
	}
	if model.inputs[0].SessionID != "session-1" || model.inputs[0].UserMessage != "你好" {
		t.Fatalf("expected model input to contain session and user message, got %+v", model.inputs[0])
	}
	if len(model.inputs[0].Tools) != 1 || model.inputs[0].Tools[0].Name != "get_game_state" {
		t.Fatalf("expected model to receive tool specs, got %+v", model.inputs[0].Tools)
	}
	if len(executor.calls) != 0 {
		t.Fatalf("expected executor not to be called, got %+v", executor.calls)
	}
}

func TestRuntimeRunExecutesToolAndThenReturnsReply(t *testing.T) {
	model := &fakeModelAdapter{
		outputs: []ModelOutput{
			{
				Thought: "先读取当前状态。",
				ToolRequest: &ToolRequest{
					Name:  "get_game_state",
					Input: json.RawMessage(`{"include_inventory":true}`),
				},
			},
			{
				Reply: "你当前背包里有一瓶治疗药水。",
			},
		},
	}
	registry := &fakeRegistry{
		specs: []tools.ToolSpec{
			{Name: "get_game_state", Description: "读取游戏进度"},
		},
	}
	executor := &fakeExecutor{
		outputs: []tools.CallOutput{
			{
				ToolName: "get_game_state",
				Content: map[string]any{
					"inventory_count": 1,
				},
			},
		},
	}

	runtime := NewRuntime(model, registry, executor)
	output, err := runtime.Run(context.Background(), RuntimeInput{
		SessionID:   "session-1",
		UserMessage: "我背包里有什么？",
		MaxSteps:    4,
	})
	if err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}
	if output.Reply != "你当前背包里有一瓶治疗药水。" {
		t.Fatalf("expected final reply, got %q", output.Reply)
	}
	if len(output.Steps) != 1 {
		t.Fatalf("expected 1 step, got %+v", output.Steps)
	}
	if output.Steps[0].Thought != "先读取当前状态。" || output.Steps[0].ActionName != "get_game_state" {
		t.Fatalf("expected step to record thought and tool name, got %+v", output.Steps[0])
	}
	if len(executor.calls) != 1 {
		t.Fatalf("expected executor to be called once, got %d", len(executor.calls))
	}
	if executor.calls[0].ToolName != "get_game_state" || executor.calls[0].Input.SessionID != "session-1" {
		t.Fatalf("expected executor call to contain tool name and session id, got %+v", executor.calls[0])
	}
	if len(model.inputs) != 2 {
		t.Fatalf("expected model to be called twice, got %d", len(model.inputs))
	}
	if len(model.inputs[1].Steps) != 1 {
		t.Fatalf("expected second model call to receive one step, got %+v", model.inputs[1].Steps)
	}
}

func TestRuntimeRunAccumulatesStepsAcrossToolCalls(t *testing.T) {
	model := &fakeModelAdapter{
		outputs: []ModelOutput{
			{
				Thought: "先查设定。",
				ToolRequest: &ToolRequest{
					Name:  "search_lore",
					Input: json.RawMessage(`{"query":"月影森林"}`),
				},
			},
			{
				Thought: "再查当前状态。",
				ToolRequest: &ToolRequest{
					Name:  "get_game_state",
					Input: json.RawMessage(`{}`),
				},
			},
			{
				Reply: "你已经确认了传说与当前状态。",
			},
		},
	}
	registry := &fakeRegistry{
		specs: []tools.ToolSpec{
			{Name: "search_lore", Description: "检索设定"},
			{Name: "get_game_state", Description: "读取游戏进度"},
		},
	}
	executor := &fakeExecutor{
		outputs: []tools.CallOutput{
			{ToolName: "search_lore", Content: map[string]any{"hits": 2}},
			{ToolName: "get_game_state", Content: map[string]any{"scene": "forest"}},
		},
	}

	runtime := NewRuntime(model, registry, executor)
	output, err := runtime.Run(context.Background(), RuntimeInput{
		SessionID:   "session-1",
		UserMessage: "月影森林和我当前状态有什么关系？",
		MaxSteps:    4,
	})
	if err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}
	if len(output.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %+v", output.Steps)
	}
	if len(model.inputs) != 3 {
		t.Fatalf("expected model to be called 3 times, got %d", len(model.inputs))
	}
	if len(model.inputs[1].Steps) != 1 || len(model.inputs[2].Steps) != 2 {
		t.Fatalf("expected model step history to accumulate, got second=%d third=%d", len(model.inputs[1].Steps), len(model.inputs[2].Steps))
	}
}

func TestRuntimeRunReturnsStepLimitExceeded(t *testing.T) {
	model := &fakeModelAdapter{
		outputs: []ModelOutput{
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
			{ToolRequest: &ToolRequest{Name: "search_rules", Input: json.RawMessage(`{"query":"潜行"}`)}},
		},
	}
	runtime := NewRuntime(model, &fakeRegistry{
		specs: []tools.ToolSpec{{Name: "search_rules", Description: "检索规则"}},
	}, &fakeExecutor{
		outputs: []tools.CallOutput{
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
			{ToolName: "search_rules", Content: map[string]any{"hits": 1}},
		},
	})

	_, err := runtime.Run(context.Background(), RuntimeInput{
		SessionID:   "session-1",
		UserMessage: "潜行规则是什么？",
	})
	if !errors.Is(err, ErrStepLimitExceeded) {
		t.Fatalf("expected ErrStepLimitExceeded, got %v", err)
	}
	if len(model.inputs) != 8 {
		t.Fatalf("expected default max steps to be 8, got %d model calls", len(model.inputs))
	}
}

func TestNormalizeRuntimeInputUsesLargerDefaultContextLimit(t *testing.T) {
	normalized := normalizeRuntimeInput(RuntimeInput{
		SessionID:   "session-1",
		UserMessage: "当前设定是什么？",
	})

	if normalized.ContextLimit != 40 {
		t.Fatalf("expected default context limit 40, got %d", normalized.ContextLimit)
	}
}

type fakeModelAdapter struct {
	outputs []ModelOutput
	inputs  []ModelInput
	err     error
}

func (f *fakeModelAdapter) Run(ctx context.Context, input ModelInput) (ModelOutput, error) {
	_ = ctx
	f.inputs = append(f.inputs, input)
	if f.err != nil {
		return ModelOutput{}, f.err
	}
	index := len(f.inputs) - 1
	if index >= len(f.outputs) {
		return ModelOutput{}, ErrInvalidModelOutput
	}
	return f.outputs[index], nil
}

type fakeRegistry struct {
	specs []tools.ToolSpec
}

func (f *fakeRegistry) Register(tool tools.Tool) error {
	_ = tool
	return nil
}

func (f *fakeRegistry) Get(name string) (tools.Tool, bool) {
	_ = name
	return nil, false
}

func (f *fakeRegistry) List() []tools.ToolSpec {
	return f.specs
}

type executorCall struct {
	ToolName string
	Input    tools.CallInput
}

type fakeExecutor struct {
	calls   []executorCall
	outputs []tools.CallOutput
	err     error
}

func (f *fakeExecutor) Execute(ctx context.Context, toolName string, input tools.CallInput) (tools.CallOutput, error) {
	_ = ctx
	f.calls = append(f.calls, executorCall{ToolName: toolName, Input: input})
	if f.err != nil {
		return tools.CallOutput{}, f.err
	}
	index := len(f.calls) - 1
	if index >= len(f.outputs) {
		return tools.CallOutput{}, nil
	}
	return f.outputs[index], nil
}
