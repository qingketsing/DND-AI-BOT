package queue

import (
	"context"
	"testing"
	"time"
)

func TestRabbitMQPublisherPublishesUsingConfiguredTopology(t *testing.T) {
	channel := &fakeRabbitMQPublisherChannel{}
	publisher := NewRabbitMQPublisher(channel)
	payload := MessageJobPayload{
		JobID:     "job-1",
		MessageID: "msg-1",
		SessionID: "session-1",
		UserID:    "user-1",
		Attempt:   1,
		QueuedAt:  time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
	}

	if err := publisher.Publish(context.Background(), payload); err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	if channel.exchangeName != MessageExchange {
		t.Fatalf("expected exchange %q, got %q", MessageExchange, channel.exchangeName)
	}
	if channel.queueName != MessageQueue {
		t.Fatalf("expected queue %q, got %q", MessageQueue, channel.queueName)
	}
	if channel.boundQueue != MessageQueue {
		t.Fatalf("expected bound queue %q, got %q", MessageQueue, channel.boundQueue)
	}
	if channel.boundExchange != MessageExchange {
		t.Fatalf("expected bound exchange %q, got %q", MessageExchange, channel.boundExchange)
	}
	if channel.boundRoutingKey != MessageRoutingKey {
		t.Fatalf("expected bound routing key %q, got %q", MessageRoutingKey, channel.boundRoutingKey)
	}
	if channel.publishExchange != MessageExchange {
		t.Fatalf("expected publish exchange %q, got %q", MessageExchange, channel.publishExchange)
	}
	if channel.publishRoutingKey != MessageRoutingKey {
		t.Fatalf("expected publish routing key %q, got %q", MessageRoutingKey, channel.publishRoutingKey)
	}

	got, err := DecodeMessageJobPayload(channel.publishBody)
	if err != nil {
		t.Fatalf("expected publish body to be decodable, got %v", err)
	}
	if got != payload {
		t.Fatalf("expected payload %+v, got %+v", payload, got)
	}
}

type fakeRabbitMQPublisherChannel struct {
	exchangeName      string
	queueName         string
	boundQueue        string
	boundRoutingKey   string
	boundExchange     string
	publishExchange   string
	publishRoutingKey string
	publishBody       []byte
}

func (f *fakeRabbitMQPublisherChannel) ExchangeDeclare(name string) error {
	f.exchangeName = name
	return nil
}

func (f *fakeRabbitMQPublisherChannel) QueueDeclare(name string) error {
	f.queueName = name
	return nil
}

func (f *fakeRabbitMQPublisherChannel) QueueBind(queue string, routingKey string, exchange string) error {
	f.boundQueue = queue
	f.boundRoutingKey = routingKey
	f.boundExchange = exchange
	return nil
}

func (f *fakeRabbitMQPublisherChannel) Publish(ctx context.Context, exchange string, routingKey string, body []byte) error {
	_ = ctx
	f.publishExchange = exchange
	f.publishRoutingKey = routingKey
	f.publishBody = append([]byte(nil), body...)
	return nil
}
