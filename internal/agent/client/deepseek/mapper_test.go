package deepseek

import (
	"encoding/json"
	"testing"

	"DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/agent/tools"
)

func TestBuildChatRequestBuildsMessagesAndTools(t *testing.T) {
	request := BuildChatRequest("deepseek-chat", runtime.ModelInput{
		SessionID:    "session-1",
		SystemPrompt: "system prompt",
		UserMessage:  "user message",
		Tools: []tools.ToolSpec{
			{
				Name:        "get_game_state",
				Description: "get state",
				InputSchema: map[string]any{"type": "object"},
			},
		},
		Steps: []runtime.StepRecord{
			{
				Thought:          "need tool",
				ReasoningContent: "private reasoning",
				ActionName:       "get_game_state",
				ActionInput:      json.RawMessage(`{"foo":"bar"}`),
				Observation:      map[string]any{"gold": 10},
			},
		},
	})

	if request.Model != "deepseek-chat" {
		t.Fatalf("expected model to be propagated, got %s", request.Model)
	}
	if len(request.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(request.Messages))
	}
	if request.Messages[0].Role != "system" {
		t.Fatalf("expected first message role to be system, got %s", request.Messages[0].Role)
	}
	if request.Messages[2].ReasoningContent != "private reasoning" {
		t.Fatalf("expected reasoning content to be propagated, got %q", request.Messages[2].ReasoningContent)
	}
	if len(request.Tools) != 1 || request.Tools[0].Function.Name != "get_game_state" {
		t.Fatalf("expected tool spec to be mapped into request, got %+v", request.Tools)
	}
}

func TestParseChatResponseReturnsReply(t *testing.T) {
	output, err := ParseChatResponse(ChatResponse{
		Choices: []ChatChoice{
			{
				Message: ChatMessage{
					Role:             "assistant",
					Content:          "final reply",
					ReasoningContent: "reply reasoning",
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
	if output.ReasoningContent != "reply reasoning" {
		t.Fatalf("expected reasoning content to be preserved, got %q", output.ReasoningContent)
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
					Role:             "assistant",
					Content:          "need tool",
					ReasoningContent: "tool reasoning",
					ToolCalls: []ChatToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: ChatToolCallFunction{
								Name:      "get_game_state",
								Arguments: `{"amount":1}`,
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
	if output.ToolRequest.Name != "get_game_state" {
		t.Fatalf("expected tool name to be propagated, got %s", output.ToolRequest.Name)
	}
	if string(output.ToolRequest.Input) != `{"amount":1}` {
		t.Fatalf("expected tool input to be propagated, got %s", string(output.ToolRequest.Input))
	}
	if output.ReasoningContent != "tool reasoning" {
		t.Fatalf("expected reasoning content to be propagated, got %q", output.ReasoningContent)
	}
}
