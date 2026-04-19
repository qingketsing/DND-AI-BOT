package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewLoggerWritesJSON(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := NewLogger("prod", buffer)

	logger.Info("agent run", slog.String("session_id", "session-1"))

	var payload map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON log output, got %v: %s", err, buffer.String())
	}
	if payload["msg"] != "agent run" || payload["session_id"] != "session-1" {
		t.Fatalf("expected structured fields in log, got %+v", payload)
	}
}

func TestRequestIDContextRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-1")

	if got := RequestIDFromContext(ctx); got != "req-1" {
		t.Fatalf("expected request id req-1, got %q", got)
	}
}
