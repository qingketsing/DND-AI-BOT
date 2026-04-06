package client

import (
	"context"

	"DND-AI-BOT/internal/agent/runtime"
)

// ModelAdapter 定义多模型厂商共享的统一适配接口。
type ModelAdapter interface {
	Run(ctx context.Context, input runtime.ModelInput) (runtime.ModelOutput, error)
}
