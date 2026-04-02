package memory

import "errors"

var (
	// ErrSessionNotFound 表示仓库中不存在目标会话。
	ErrSessionNotFound = errors.New("session not found")
	// ErrNilSession 表示调用方传入了空会话。
	ErrNilSession = errors.New("session is nil")
	// ErrEmptySessionID 表示会话 ID 为空，无法保存或读取。
	ErrEmptySessionID = errors.New("session id is empty")
)
