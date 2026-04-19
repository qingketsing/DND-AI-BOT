package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type requestIDContextKey struct{}

// NewLogger 创建 JSON 结构化 logger。
func NewLogger(env string, output io.Writer) *slog.Logger {
	_ = env
	if output == nil {
		output = os.Stdout
	}
	return slog.New(slog.NewJSONHandler(output, nil))
}

// DefaultLogger 返回默认 JSON logger。
func DefaultLogger() *slog.Logger {
	return NewLogger(os.Getenv("APP_ENV"), os.Stdout)
}

// WithRequestID 将 request id 写入 context。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext 从 context 读取 request id。
func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}
