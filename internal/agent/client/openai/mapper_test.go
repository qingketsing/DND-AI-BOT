package openai

import (
	"encoding/json"
	"testing"

	"DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/agent/tools"
)

func TestBuildChatRequestBuildsMessagesAndTools(t *testing.T) {
	request := BuildChatRequest("gpt-5.4-mini", runtime.ModelInput{
		SessionID:    "session-1",
		SystemPrompt: "system prompt",
		UserMessage:  "user message",
		Tools: []tools.ToolSpec{
			{
				Name:        "apply_damage",
				Description: "apply damage",
				InputSchema: map[string]any{"type": "object"},
			},
		},
		Steps: []runtime.StepRecord{
			{
				Thought:     "need tool",
				ActionName:  "apply_damage",
				ActionInput: json.RawMessage(`{"amount":3}`),
				Observation: map[string]any{"hp": 7},
			},
		},
	})

	if request.Model != "gpt-5.4-mini" {
		t.Fatalf("expected model to be propagated, got %s", request.Model)
	}
	if len(request.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(request.Messages))
	}
	if request.Messages[1].Role != "user" {
		t.Fatalf("expected second message role to be user, got %s", request.Messages[1].Role)
	}
	if len(request.Tools) != 1 || request.Tools[0].Function.Name != "apply_damage" {
		t.Fatalf("expected tool spec to be mapped into request, got %+v", request.Tools)
	}
}

func TestParseChatResponseReturnsReply(t *testing.T) {
	output, err := ParseChatResponse(ChatResponse{
		Choices: []ChatChoice{
			{
				Message: ChatMessage{
					Role:    "assistant",
					Content: "final reply",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected reply response to parse, got %v", err)
	}
	if output.Reply != "final reply" {
		t.Fatalf("expected final reply, got %s", output.Reply)
	}
	if output.ToolRequest != nil {
		t.Fatal("expected no tool request in reply response")
	}
}

func TestParseChatResponseReturnsToolRequest(t *testing.T) {
	output, err := ParseChatResponse(ChatResponse{
		Choices: []ChatChoice{
			{
				Message: ChatMessage{
					Role:    "assistant",
					Content: "need tool",
					ToolCalls: []ChatToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: ChatToolCallFunction{
								Name:      "apply_damage",
								Arguments: `{"amount":3}`,
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected tool response to parse, got %v", err)
	}
	if output.ToolRequest == nil {
		t.Fatal("expected tool request to be present")
	}
	if output.ToolRequest.Name != "apply_damage" {
		t.Fatalf("expected tool name to be propagated, got %s", output.ToolRequest.Name)
	}
	if string(output.ToolRequest.Input) != `{"amount":3}` {
		t.Fatalf("expected tool input to be propagated, got %s", string(output.ToolRequest.Input))
	}
}
