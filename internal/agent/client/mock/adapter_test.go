package mock

import (
	"context"
	"errors"
	"testing"

	"DND-AI-BOT/internal/agent/runtime"
)

func TestAdapterRunReturnsOutputsInOrder(t *testing.T) {
	adapter := NewAdapter([]runtime.ModelOutput{
		{Reply: "first"},
		{Reply: "second"},
	})

	first, err := adapter.Run(context.Background(), runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	})
	if err != nil {
		t.Fatalf("expected first run to succeed, got %v", err)
	}
	if first.Reply != "first" {
		t.Fatalf("expected first reply %q, got %q", "first", first.Reply)
	}

	second, err := adapter.Run(context.Background(), runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "world",
	})
	if err != nil {
		t.Fatalf("expected second run to succeed, got %v", err)
	}
	if second.Reply != "second" {
		t.Fatalf("expected second reply %q, got %q", "second", second.Reply)
	}
}

func TestAdapterRunReturnsErrorWhenOutputsExhausted(t *testing.T) {
	adapter := NewAdapter([]runtime.ModelOutput{
		{Reply: "only-once"},
	})

	if _, err := adapter.Run(context.Background(), runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "hello",
	}); err != nil {
		t.Fatalf("expected first run to succeed, got %v", err)
	}

	_, err := adapter.Run(context.Background(), runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "again",
	})
	if !errors.Is(err, ErrNoMoreMockOutputs) {
		t.Fatalf("expected ErrNoMoreMockOutputs, got %v", err)
	}
}

func TestAdapterInputsRecordsRunInputs(t *testing.T) {
	adapter := NewAdapter([]runtime.ModelOutput{
		{Reply: "done"},
	})

	input := runtime.ModelInput{
		SessionID:   "session-1",
		UserMessage: "inspect me",
	}

	if _, err := adapter.Run(context.Background(), input); err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}

	inputs := adapter.Inputs()
	if len(inputs) != 1 {
		t.Fatalf("expected 1 recorded input, got %d", len(inputs))
	}
	if inputs[0].SessionID != "session-1" || inputs[0].UserMessage != "inspect me" {
		t.Fatalf("expected recorded input to match, got %+v", inputs[0])
	}
}
