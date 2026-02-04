package ai

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

// TestChatConnection 需要在项目根目录有 .env 文件且配置了有效的 API Key
// 运行方法: go test -v ./pkg/ai/... -run TestChatConnection
func TestChatConnection(t *testing.T) {
	// 加载根目录的 .env
	// 注意：测试运行时当前目录是 pkg/ai，所以 .env 在 ../../.env
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Log("Warning: .env file not found, trying system env")
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set")
	}

	InitAI()

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个测试助手。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "如果你能听到我说话，请回复 'Roger'.",
		},
	}

	response, err := GlobalClient.ChatRequest(context.Background(), messages)
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}

	t.Logf("AI Response: %s", response)
	logrus.Infof("AI Response received successfully")
}
