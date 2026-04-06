package deepseek

import (
	"encoding/json"
	"fmt"

	"DND-AI-BOT/internal/agent/runtime"
)

// BuildChatRequest 将统一模型输入转换为 DeepSeek 的聊天补全请求。
func BuildChatRequest(model string, input runtime.ModelInput) ChatRequest {
	messages := make([]ChatMessage, 0, len(input.Steps)*2+2)
	if input.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: input.SystemPrompt,
		})
	}
	if input.UserMessage != "" {
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: input.UserMessage,
		})
	}

	for index, step := range input.Steps {
		if step.ActionName != "" {
			toolCallID := fmt.Sprintf("call_%d", index+1)
			messages = append(messages, ChatMessage{
				Role:    "assistant",
				Content: step.Thought,
				ToolCalls: []ChatToolCall{
					{
						ID:   toolCallID,
						Type: "function",
						Function: ChatToolCallFunction{
							Name:      step.ActionName,
							Arguments: string(step.ActionInput),
						},
					},
				},
			})
			messages = append(messages, ChatMessage{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    marshalObservation(step.Observation),
			})
			continue
		}

		if step.Thought != "" {
			messages = append(messages, ChatMessage{
				Role:    "assistant",
				Content: step.Thought,
			})
		}
	}

	tools := make([]ChatTool, 0, len(input.Tools))
	for _, tool := range input.Tools {
		tools = append(tools, ChatTool{
			Type: "function",
			Function: ChatToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}
}

// ParseChatResponse 将 DeepSeek 聊天补全响应转换为 Runtime 输出。
func ParseChatResponse(response ChatResponse) (runtime.ModelOutput, error) {
	if len(response.Choices) == 0 {
		return runtime.ModelOutput{}, fmt.Errorf("deepseek response has no choices")
	}

	message := response.Choices[0].Message
	if len(message.ToolCalls) > 0 {
		call := message.ToolCalls[0]
		return runtime.ModelOutput{
			Thought: message.Content,
			ToolRequest: &runtime.ToolRequest{
				Name:  call.Function.Name,
				Input: json.RawMessage(call.Function.Arguments),
			},
		}, nil
	}

	if message.Content == "" {
		return runtime.ModelOutput{}, fmt.Errorf("deepseek response content is empty")
	}

	return runtime.ModelOutput{
		Reply: message.Content,
	}, nil
}

func marshalObservation(observation any) string {
	if observation == nil {
		return "null"
	}

	data, err := json.Marshal(observation)
	if err != nil {
		return `{"error":"failed to marshal observation"}`
	}

	return string(data)
}
