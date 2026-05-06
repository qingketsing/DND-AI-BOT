package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	MessageExchange   = "agent.message"
	MessageQueue      = "agent.message.default"
	MessageRoutingKey = "message.process"
)

type MessageJobPayload struct {
	JobID     string    `json:"job_id"`
	MessageID string    `json:"message_id"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Attempt   int       `json:"attempt"`
	QueuedAt  time.Time `json:"queued_at"`
}

type MessageJobPublisher interface {
	Publish(ctx context.Context, payload MessageJobPayload) error
}

type MessageJobConsumer interface {
	Receive(ctx context.Context, handler func(context.Context, MessageJobPayload) error) error
}

func EncodeMessageJobPayload(payload MessageJobPayload) ([]byte, error) {
	return json.Marshal(payload)
}

func DecodeMessageJobPayload(raw []byte) (MessageJobPayload, error) {
	var payload MessageJobPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return MessageJobPayload{}, err
	}
	return payload, nil
}

type InMemoryTransport struct {
	queue []MessageJobPayload
}

func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{queue: make([]MessageJobPayload, 0)}
}

type InMemoryPublisher struct {
	transport *InMemoryTransport
}

func NewInMemoryPublisher(transport *InMemoryTransport) *InMemoryPublisher {
	return &InMemoryPublisher{transport: transport}
}

func (p *InMemoryPublisher) Publish(ctx context.Context, payload MessageJobPayload) error {
	if p.transport == nil {
		return errors.New("in-memory transport is nil")
	}
	p.transport.queue = append(p.transport.queue, payload)
	return nil
}

type InMemoryConsumer struct {
	transport *InMemoryTransport
}

func NewInMemoryConsumer(transport *InMemoryTransport) *InMemoryConsumer {
	return &InMemoryConsumer{transport: transport}
}

func (c *InMemoryConsumer) Receive(ctx context.Context, handler func(context.Context, MessageJobPayload) error) error {
	if c.transport == nil {
		return errors.New("in-memory transport is nil")
	}
	if len(c.transport.queue) == 0 {
		return errors.New("in-memory queue is empty")
	}
	payload := c.transport.queue[0]
	c.transport.queue = c.transport.queue[1:]
	return handler(ctx, payload)
}
