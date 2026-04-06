package client

import "errors"

var (
	// ErrUnsupportedProvider 表示当前厂商类型尚未实现。
	ErrUnsupportedProvider = errors.New("unsupported provider")
	// ErrInvalidClientConfig 表示模型客户端配置不合法。
	ErrInvalidClientConfig = errors.New("invalid client config")
	// ErrModelRequestFailed 表示对模型的请求执行失败。
	ErrModelRequestFailed = errors.New("model request failed")
	// ErrInvalidModelResponse 表示模型返回内容不符合约定。
	ErrInvalidModelResponse = errors.New("invalid model response")
)
