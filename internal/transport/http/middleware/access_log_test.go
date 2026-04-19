package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"DND-AI-BOT/internal/observability"
)

func TestAccessLogMiddlewareWritesStructuredLog(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewJSONHandler(buffer, nil))
	handler := NewAccessLogMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	request := httptest.NewRequest(http.MethodPost, "/sessions", nil)
	request = request.WithContext(observability.WithRequestID(request.Context(), "req-1"))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var payload map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON access log, got %v: %s", err, buffer.String())
	}
	if payload["msg"] != "http request" || payload["method"] != http.MethodPost || payload["status"] != float64(http.StatusCreated) || payload["request_id"] != "req-1" {
		t.Fatalf("unexpected access log payload %+v", payload)
	}
}
