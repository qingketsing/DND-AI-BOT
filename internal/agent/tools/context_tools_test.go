package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	agentcontext "DND-AI-BOT/internal/agent/context"
	"DND-AI-BOT/internal/model"
)

func TestGetAgentContextToolCallBuildsContext(t *testing.T) {
	now := time.Date(2026, 4, 6, 14, 0, 0, 0, time.UTC)
	provider := &fakeContextProvider{
		result: agentcontext.AgentContext{
			SessionID: "session-1",
			Channel:   model.ChannelWeb,
		},
	}
	tool := NewGetAgentContextTool(provider)

	output, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"limit":8}`),
		Now:       now,
	})
	if err != nil {
		t.Fatalf("expected call to succeed, got %v", err)
	}
	if output.ToolName != "get_agent_context" {
		t.Fatalf("expected tool name %q, got %q", "get_agent_context", output.ToolName)
	}
	if provider.limit != 8 {
		t.Fatalf("expected limit 8, got %d", provider.limit)
	}
	result, ok := output.Content.(GetAgentContextResult)
	if !ok {
		t.Fatalf("expected GetAgentContextResult, got %T", output.Content)
	}
	if result.SessionID != "session-1" || result.Channel != "web" {
		t.Fatalf("expected mapped context result, got %+v", result)
	}
}

func TestGetAgentContextToolCallRejectsInvalidInput(t *testing.T) {
	tool := NewGetAgentContextTool(&fakeContextProvider{})

	_, err := tool.Call(context.Background(), CallInput{
		SessionID: "session-1",
		Raw:       json.RawMessage(`{"limit":"bad"}`),
	})
	if !errors.Is(err, ErrInvalidToolInput) {
		t.Fatalf("expected ErrInvalidToolInput, got %v", err)
	}
}

type fakeContextProvider struct {
	result    agentcontext.AgentContext
	err       error
	sessionID string
	limit     int
}

func (f *fakeContextProvider) BuildContext(ctx context.Context, sessionID string, limit int) (agentcontext.AgentContext, error) {
	_ = ctx
	f.sessionID = sessionID
	f.limit = limit
	if f.err != nil {
		return agentcontext.AgentContext{}, f.err
	}
	return f.result, nil
}
