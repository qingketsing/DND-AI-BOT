package service

import (
	"context"
	"strings"
	"testing"
)

func TestLLMSessionSummarizerSummarizeMessagesParsesJSONReply(t *testing.T) {
	model := &fakeSummaryModel{
		reply: `{"character_summary":"青稞，精灵法师。","scene_summary":"位于 the city 广场。","current_objective":"寻找格伦。","recent_key_events":["创建角色","查看布告栏"]}`,
	}
	summarizer := NewLLMSessionSummarizer(model)

	result, err := summarizer.SummarizeMessages(context.Background(), "session-1", []SummarizerMessage{
		{Source: "user", User: "Qingke", Content: "我想创建一个精灵法师角色"},
		{Source: "agent", User: "DM Agent", Content: "你现在位于 the city 广场"},
	})
	if err != nil {
		t.Fatalf("expected summarize to succeed, got %v", err)
	}
	if result.CharacterSummary != "青稞，精灵法师。" || result.SceneSummary != "位于 the city 广场。" {
		t.Fatalf("unexpected summary result %+v", result)
	}
	if len(result.RecentKeyEvents) != 2 {
		t.Fatalf("expected key events to be parsed, got %+v", result.RecentKeyEvents)
	}

	if model.calls != 1 {
		t.Fatalf("expected one model call, got %d", model.calls)
	}
	if !strings.Contains(model.userPrompt, "Qingke") || !strings.Contains(model.userPrompt, "the city 广场") {
		t.Fatalf("expected transcript content in summarizer input, got %q", model.userPrompt)
	}
}

func TestLLMSessionSummarizerRejectsInvalidJSON(t *testing.T) {
	summarizer := NewLLMSessionSummarizer(&fakeSummaryModel{reply: "not-json"})

	_, err := summarizer.SummarizeMessages(context.Background(), "session-1", []SummarizerMessage{
		{Source: "user", User: "Qingke", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected invalid json to fail")
	}
}

type fakeSummaryModel struct {
	reply        string
	err          error
	calls        int
	systemPrompt string
	userPrompt   string
}

func (f *fakeSummaryModel) Summarize(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	_ = ctx
	f.calls++
	f.systemPrompt = systemPrompt
	f.userPrompt = userPrompt
	return f.reply, f.err
}
