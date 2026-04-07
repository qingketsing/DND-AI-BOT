package mock

import (
	"context"
	"testing"

	"DND-AI-BOT/internal/agent/runtime"
)

func TestEchoAdapterRunReturnsReplyFromUserMessage(t *testing.T) {
	adapter := NewEchoAdapter()

	output, err := adapter.Run(context.Background(), runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected echo adapter to succeed, got %v", err)
	}
	if output.Reply != "mock reply: hello" {
		t.Fatalf("expected echo reply, got %q", output.Reply)
	}
	if output.ToolRequest != nil {
		t.Fatal("expected echo adapter to return final reply without tool request")
	}
}
