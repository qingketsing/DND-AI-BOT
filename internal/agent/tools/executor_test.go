package tools

import (
	"context"
	"errors"
	"net/http/httptest"
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

func TestDefaultExecutorExecuteRunsTargetTool(t *testing.T) {
	registry := NewInMemoryRegistry()
	tool := executorStubTool{
		name: "get_agent_context",
		output: CallOutput{
			ToolName: "get_agent_context",
			Content: map[string]any{
				"session_id": "session-1",
			},
		},
	}
	if err := registry.Register(tool); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry)
	output, err := executor.Execute(context.Background(), "get_agent_context", CallInput{
		SessionID: "session-1",
		Now:       time.Date(2026, 4, 6, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected execute to succeed, got %v", err)
	}
	if output.ToolName != "get_agent_context" {
		t.Fatalf("expected tool name %q, got %q", "get_agent_context", output.ToolName)
	}
	if tool.calledCount != 0 {
		t.Fatal("expected original stub copy not to be mutated")
	}
}

func TestDefaultExecutorExecuteReturnsToolNotFound(t *testing.T) {
	executor := NewExecutor(NewInMemoryRegistry())

	_, err := executor.Execute(context.Background(), "missing_tool", CallInput{})
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}

func TestDefaultExecutorExecutePropagatesToolError(t *testing.T) {
	registry := NewInMemoryRegistry()
	expectedErr := errors.New("boom")
	if err := registry.Register(executorStubTool{
		name: "spend_gold",
		err:  expectedErr,
	}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry)
	_, err := executor.Execute(context.Background(), "spend_gold", CallInput{
		SessionID: "session-1",
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected propagated tool error, got %v", err)
	}
}

func TestDefaultExecutorExecuteRecordsSuccessMetrics(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	registry := NewInMemoryRegistry()
	if err := registry.Register(executorStubTool{
		name:   "get_agent_context",
		output: CallOutput{ToolName: "get_agent_context"},
	}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry, WithExecutorMetrics(metrics))
	_, err := executor.Execute(context.Background(), "get_agent_context", CallInput{})
	if err != nil {
		t.Fatalf("expected execute to succeed, got %v", err)
	}

	body := scrapeToolMetrics(t, metrics)
	if !strings.Contains(body, `tool_calls_total{status="success",tool="get_agent_context"} 1`) {
		t.Fatalf("expected successful tool metric, got %s", body)
	}
}

func TestDefaultExecutorExecuteRecordsErrorMetrics(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	registry := NewInMemoryRegistry()
	if err := registry.Register(executorStubTool{
		name: "spend_gold",
		err:  errors.New("boom"),
	}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry, WithExecutorMetrics(metrics))
	_, err := executor.Execute(context.Background(), "spend_gold", CallInput{})
	if err == nil {
		t.Fatal("expected execute to fail")
	}

	body := scrapeToolMetrics(t, metrics)
	if !strings.Contains(body, `tool_calls_total{status="error",tool="spend_gold"} 1`) ||
		!strings.Contains(body, `tool_errors_total{tool="spend_gold"} 1`) {
		t.Fatalf("expected error tool metrics, got %s", body)
	}
}

func TestDefaultExecutorExecuteLogsToolFailure(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	registry := NewInMemoryRegistry()
	if err := registry.Register(executorStubTool{
		name: "spend_gold",
		err:  errors.New("boom"),
	}); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	executor := NewExecutor(registry, WithExecutorLogger(logger))
	_, err := executor.Execute(context.Background(), "spend_gold", CallInput{
		SessionID: "session-1",
	})
	if err == nil {
		t.Fatal("expected execute to fail")
	}

	logOutput := buffer.String()
	for _, expected := range []string{"tool execution failed", "tool=spend_gold", "session_id=session-1", "error=boom"} {
		if !strings.Contains(logOutput, expected) {
			t.Fatalf("expected log output to contain %q, got %s", expected, logOutput)
		}
	}
}

func scrapeToolMetrics(t *testing.T, metrics *observability.Metrics) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	return recorder.Body.String()
}

type executorStubTool struct {
	name        string
	output      CallOutput
	err         error
	calledCount int
}

func (t executorStubTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: "执行器测试桩工具",
		InputSchema: map[string]any{
			"type": "object",
		},
	}
}

func (t executorStubTool) Call(ctx context.Context, input CallInput) (CallOutput, error) {
	_ = ctx
	_ = input
	if t.err != nil {
		return CallOutput{}, t.err
	}
	return t.output, nil
}
