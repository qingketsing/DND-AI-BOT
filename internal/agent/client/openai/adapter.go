package openai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"DND-AI-BOT/internal/agent/runtime"
)

// Adapter 将 Runtime 的统一模型输入映射到 OpenAI 的聊天补全协议。
type Adapter struct {
	client *Client
	model  string
}

// NewAdapter 创建一个基于 OpenAI 的模型适配器。
func NewAdapter(model string, baseURL string, apiKey string, timeout time.Duration) (*Adapter, error) {
	httpClient := &http.Client{Timeout: timeoutOrDefault(timeout)}

	return &Adapter{
		client: NewClient(baseURL, apiKey, httpClient),
		model:  model,
	}, nil
}

// Run 执行一次统一模型请求，并将 OpenAI 响应映射回 Runtime 输出。
func (a *Adapter) Run(ctx context.Context, input runtime.ModelInput) (runtime.ModelOutput, error) {
	request := BuildChatRequest(a.model, input)
	response, err := a.client.ChatCompletion(ctx, request)
	if err != nil {
		return runtime.ModelOutput{}, fmt.Errorf("openai chat completion: %w", err)
	}

	return ParseChatResponse(response)
}

func timeoutOrDefault(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 60 * time.Second
	}
	return timeout
}
