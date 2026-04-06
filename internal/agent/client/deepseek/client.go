package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.deepseek.com"

// HTTPClient 定义底层 HTTP 客户端的最小能力，便于注入测试替身。
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ChatRequest 定义发往 DeepSeek 聊天补全接口的最小请求结构。
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []ChatTool    `json:"tools,omitempty"`
}

// ChatMessage 表示聊天消息或工具调用结果。
type ChatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`
}

// ChatTool 定义发给模型的单个工具描述。
type ChatTool struct {
	Type     string           `json:"type"`
	Function ChatToolFunction `json:"function"`
}

// ChatToolFunction 定义单个工具的函数元信息。
type ChatToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatToolCall 表示模型返回的工具调用请求。
type ChatToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function ChatToolCallFunction `json:"function"`
}

// ChatToolCallFunction 表示模型发起的具体函数调用。
type ChatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse 定义 DeepSeek 聊天补全接口的最小响应结构。
type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
}

// ChatChoice 表示一个模型候选输出。
type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

// Client 封装 DeepSeek 聊天补全请求所需的底层 HTTP 细节。
type Client struct {
	baseURL    string
	apiKey     string
	httpClient HTTPClient
}

// NewClient 创建一个可复用的 DeepSeek HTTP 客户端。
func NewClient(baseURL string, apiKey string, httpClient HTTPClient) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// ChatCompletion 调用 DeepSeek 的聊天补全接口并返回解码后的响应。
func (c *Client) ChatCompletion(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	var response ChatResponse

	payload, err := json.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("marshal deepseek request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return response, fmt.Errorf("build deepseek request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return response, fmt.Errorf("execute deepseek request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return response, fmt.Errorf("deepseek request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return response, fmt.Errorf("decode deepseek response: %w", err)
	}

	return response, nil
}
