package service

import (
	"bytes"
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/observability"
)

func TestAgentServiceReplyReturnsRuntimeOutput(t *testing.T) {
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{
			Reply: "你当前背包里有一瓶治疗药水。",
			Steps: []AgentStep{{ToolName: "get_game_state"}},
		}, nil
	}, nil)

	output, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "我的背包里有什么？",
		MaxSteps:    4,
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}
	if output.Reply != "你当前背包里有一瓶治疗药水。" {
		t.Fatalf("expected final reply, got %q", output.Reply)
	}
	if len(output.Steps) != 1 {
		t.Fatalf("expected one tool step, got %+v", output.Steps)
	}
}

func TestAgentServiceReplyRecordsMetrics(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{Reply: "ok"}, nil
	}, nil, WithAgentMetrics(metrics))

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}

	body := scrapeMetrics(t, metrics)
	if !strings.Contains(body, `agent_runs_total{status="success"} 1`) {
		t.Fatalf("expected successful agent run metric, got %s", body)
	}
}

func TestAgentServiceReplyRecordsFallbackMetrics(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{}, errors.New("deepseek chat completion: timeout")
	}, nil, WithAgentMetrics(metrics))

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected fallback reply, got %v", err)
	}

	body := scrapeMetrics(t, metrics)
	if !strings.Contains(body, `agent_runs_total{status="fallback"} 1`) || !strings.Contains(body, `agent_fallback_total{fallback_kind="model"} 1`) {
		t.Fatalf("expected fallback metrics, got %s", body)
	}
}

func TestAgentServiceReplyWritesStructuredLog(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	structuredLogger := slog.New(slog.NewJSONHandler(buffer, nil))
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{Reply: "ok"}, nil
	}, nil, WithAgentLogger(structuredLogger))

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, `"msg":"agent run finished"`) || !strings.Contains(logOutput, `"session_id":"session-1"`) {
		t.Fatalf("expected structured agent log, got %s", logOutput)
	}
}

func TestAgentServiceReplyLogsSuccess(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{
			Reply: "你当前背包里有一瓶治疗药水。",
			Steps: []AgentStep{{ToolName: "get_game_state"}},
		}, nil
	}, logger)

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "我的背包里有什么？",
		MaxSteps:    4,
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "agent run started") {
		t.Fatalf("expected start log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "agent run finished") {
		t.Fatalf("expected finish log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "get_game_state") {
		t.Fatalf("expected tool name to appear in logs, got %q", logOutput)
	}
}

func TestAgentServiceReplyLogsEffectiveDefaultsWhenUnset(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{
			Reply: "你好，冒险者。",
		}, nil
	}, logger)

	_, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected reply to succeed, got %v", err)
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "max_steps=8") {
		t.Fatalf("expected effective max steps in log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "context_limit=40") {
		t.Fatalf("expected effective context limit in log, got %q", logOutput)
	}
}

func TestAgentServiceReplyFallsBackWhenRunnerFails(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	logger := log.New(buffer, "", 0)
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{}, errors.New("deepseek chat completion: timeout")
	}, logger)

	output, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected reply to fall back instead of failing, got %v", err)
	}
	if !strings.Contains(output.Reply, "模型服务暂时不可用") {
		t.Fatalf("expected model fallback reply, got %q", output.Reply)
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "agent run fallback") {
		t.Fatalf("expected fallback log, got %q", logOutput)
	}
}

func TestAgentServiceReplyUsesCustomFallbackResponder(t *testing.T) {
	service := NewAgentService(func(ctx context.Context, input AgentReplyInput) (AgentReplyResult, error) {
		_ = ctx
		_ = input
		return AgentReplyResult{}, errors.New("runtime failed")
	}, nil, WithAgentFallbackResponder(stubAgentFallbackResponder{reply: "自定义兜底"}))

	output, err := service.Reply(context.Background(), AgentReplyInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected reply to fall back, got %v", err)
	}
	if output.Reply != "自定义兜底" {
		t.Fatalf("expected custom fallback reply, got %q", output.Reply)
	}
}

func TestToolNamesFromStepsReturnsCalledTools(t *testing.T) {
	names := toolNamesFromSteps([]AgentStep{
		{ToolName: "get_game_state"},
		{ToolName: ""},
		{ToolName: "skill_check"},
	})

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}
	if names[0] != "get_game_state" || names[1] != "skill_check" {
		t.Fatalf("unexpected tool names %v", names)
	}
}

func scrapeMetrics(t *testing.T, metrics *observability.Metrics) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	return recorder.Body.String()
}

type stubAgentFallbackResponder struct {
	reply string
}

func (s stubAgentFallbackResponder) BuildFallbackReply(input AgentFallbackInput) AgentFallbackReply {
	return AgentFallbackReply{
		Reply: s.reply,
		Kind:  input.Kind,
	}
}
