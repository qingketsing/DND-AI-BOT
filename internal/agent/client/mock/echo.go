package mock

import (
	"context"
	"strings"

	"DND-AI-BOT/internal/agent/runtime"
)

// EchoAdapter 是一个可直接用于本地联调的 mock 模型适配器。
// 它不会调用外部模型，而是根据用户输入返回一条稳定回复。
type EchoAdapter struct{}

// NewEchoAdapter 创建一个用于本地运行链路验证的 mock 适配器。
func NewEchoAdapter() *EchoAdapter {
	return &EchoAdapter{}
}

// Run 直接基于用户输入生成最终回复，不发起工具调用。
func (a *EchoAdapter) Run(ctx context.Context, input runtime.ModelInput) (runtime.ModelOutput, error) {
	_ = ctx

	message := strings.TrimSpace(input.UserMessage)
	if message == "" {
		message = "..."
	}

	return runtime.ModelOutput{
		Reply: "mock reply: " + message,
	}, nil
}
