package tools

import "errors"

var (
	// ErrToolNotFound 表示未找到指定名称的工具。
	ErrToolNotFound = errors.New("tool not found")
	// ErrInvalidToolInput 表示工具输入参数不合法。
	ErrInvalidToolInput = errors.New("invalid tool input")
	// ErrDuplicateTool 表示重复注册了同名工具。
	ErrDuplicateTool = errors.New("duplicate tool")
)
