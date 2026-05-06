package queue

import (
	"context"
	"testing"
	"time"
)

func TestEncodeDecodeMessageJobPayload(t *testing.T) {
	payload := MessageJobPayload{
		JobID:     "job-1",
		MessageID: "msg-1",
		SessionID: "session-1",
		UserID:    "user-1",
		Attempt:   2,
		QueuedAt:  time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
	}

	raw, err := EncodeMessageJobPayload(payload)
	if err != nil {
		t.Fatalf("expected encode to succeed, got %v", err)
	}

	got, err := DecodeMessageJobPayload(raw)
	if err != nil {
		t.Fatalf("expected decode to succeed, got %v", err)
	}
	if got != payload {
		t.Fatalf("expected payload to round trip, got %+v", got)
	}
}

func TestInMemoryMessageJobPublisherAndConsumer(t *testing.T) {
	transport := NewInMemoryTransport()
	publisher := NewInMemoryPublisher(transport)
	consumer := NewInMemoryConsumer(transport)
	ctx := context.Background()

	want := MessageJobPayload{
		JobID:     "job-2",
		MessageID: "msg-2",
		SessionID: "session-2",
		UserID:    "user-2",
		Attempt:   1,
		QueuedAt:  time.Date(2026, 5, 6, 10, 10, 0, 0, time.UTC),
	}

	if err := publisher.Publish(ctx, want); err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	var got MessageJobPayload
	if err := consumer.Receive(ctx, func(ctx context.Context, payload MessageJobPayload) error {
		got = payload
		return nil
	}); err != nil {
		t.Fatalf("expected receive to succeed, got %v", err)
	}

	if got != want {
		t.Fatalf("expected payload %+v, got %+v", want, got)
	}
}
