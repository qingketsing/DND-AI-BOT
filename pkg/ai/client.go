package ai

import (
	"context"
	"os"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

// Client 封装 OpenAI 客户端
type Client struct {
	api      *openai.Client
	model    string
	systemPM string // 系统默认 Prompt
}

var GlobalClient *Client

// InitAI 初始化全局 AI 客户端
func InitAI() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	model := os.Getenv("MODEL_NAME")

	if apiKey == "" {
		logrus.Fatal("OPENAI_API_KEY is not set")
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-chat"
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	GlobalClient = &Client{
		api:   openai.NewClientWithConfig(config),
		model: model,
	}

	logrus.Infof("AI Client initialized with Model: %s, BaseURL: %s", model, baseURL)
}

// ChatRequest 发送对话请求
// messages: 包含了历史对话列表
func (c *Client) ChatRequest(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	resp, err := c.api.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    c.model,
			Messages: messages,
		},
	)

	if err != nil {
		logrus.Errorf("ChatCompletion error: %v", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
