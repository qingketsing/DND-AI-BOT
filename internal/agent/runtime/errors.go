package runtime

import "errors"

var (
	// ErrInvalidRuntimeInput 表示 Runtime 输入参数不合法。
	ErrInvalidRuntimeInput = errors.New("invalid runtime input")
	// ErrInvalidModelOutput 表示模型返回了不符合协议的输出。
	ErrInvalidModelOutput = errors.New("invalid model output")
	// ErrStepLimitExceeded 表示本轮 ReAct 执行超过了最大步数限制。
	ErrStepLimitExceeded = errors.New("step limit exceeded")
	// ErrToolFailureLimitExceeded 表示工具连续失败次数超过上限。
	ErrToolFailureLimitExceeded = errors.New("tool failure limit exceeded")
)
