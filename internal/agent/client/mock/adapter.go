package mock

import (
	"context"
	"errors"

	"DND-AI-BOT/internal/agent/runtime"
)

var (
	// ErrNoMoreMockOutputs 表示预设的 mock 输出已经全部消费完毕。
	ErrNoMoreMockOutputs = errors.New("no more mock outputs")
)

// Adapter 是一个按预设顺序返回模型输出的脚本型 mock 模型适配器。
type Adapter struct {
	outputs []runtime.ModelOutput
	index   int
	inputs  []runtime.ModelInput
}

// NewAdapter 创建一个新的 mock 模型适配器，并拷贝保存预设输出。
func NewAdapter(outputs []runtime.ModelOutput) *Adapter {
	clonedOutputs := make([]runtime.ModelOutput, len(outputs))
	copy(clonedOutputs, outputs)

	return &Adapter{
		outputs: clonedOutputs,
		inputs:  make([]runtime.ModelInput, 0, len(outputs)),
	}
}

// Run 记录本次模型输入，并按顺序返回预设的模型输出。
func (a *Adapter) Run(ctx context.Context, input runtime.ModelInput) (runtime.ModelOutput, error) {
	_ = ctx

	a.inputs = append(a.inputs, input)
	if a.index >= len(a.outputs) {
		return runtime.ModelOutput{}, ErrNoMoreMockOutputs
	}

	output := a.outputs[a.index]
	a.index++
	return output, nil
}

// Inputs 返回当前已记录的模型输入副本，便于测试和调试。
func (a *Adapter) Inputs() []runtime.ModelInput {
	clonedInputs := make([]runtime.ModelInput, len(a.inputs))
	copy(clonedInputs, a.inputs)
	return clonedInputs
}
