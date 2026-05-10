package queue

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRabbitMQConsumerReceivesAndAcknowledgesPayload(t *testing.T) {
	payload := MessageJobPayload{
		JobID:     "job-2",
		MessageID: "msg-2",
		SessionID: "session-2",
		UserID:    "user-2",
		Attempt:   2,
		QueuedAt:  time.Date(2026, 5, 10, 10, 10, 0, 0, time.UTC),
	}
	raw, err := EncodeMessageJobPayload(payload)
	if err != nil {
		t.Fatalf("expected encode to succeed, got %v", err)
	}

	delivery := &fakeRabbitMQDelivery{body: raw}
	channel := &fakeRabbitMQConsumerChannel{
		deliveries: []*fakeRabbitMQDelivery{delivery},
	}
	consumer := NewRabbitMQConsumer(channel)

	var got MessageJobPayload
	if err := consumer.Receive(context.Background(), func(ctx context.Context, received MessageJobPayload) error {
		_ = ctx
		got = received
		return nil
	}); err != nil {
		t.Fatalf("expected receive to succeed, got %v", err)
	}

	if got != payload {
		t.Fatalf("expected payload %+v, got %+v", payload, got)
	}
	if !delivery.acked {
		t.Fatal("expected delivery to be acked")
	}
	if channel.exchangeName != MessageExchange || channel.queueName != MessageQueue || channel.boundRoutingKey != MessageRoutingKey {
		t.Fatalf("unexpected topology setup: %+v", channel)
	}
}

func TestRabbitMQConsumerNacksOnHandlerError(t *testing.T) {
	payload := MessageJobPayload{
		JobID:     "job-3",
		MessageID: "msg-3",
		SessionID: "session-3",
		UserID:    "user-3",
		Attempt:   1,
		QueuedAt:  time.Date(2026, 5, 10, 10, 20, 0, 0, time.UTC),
	}
	raw, err := EncodeMessageJobPayload(payload)
	if err != nil {
		t.Fatalf("expected encode to succeed, got %v", err)
	}

	delivery := &fakeRabbitMQDelivery{body: raw}
	channel := &fakeRabbitMQConsumerChannel{
		deliveries: []*fakeRabbitMQDelivery{delivery},
	}
	consumer := NewRabbitMQConsumer(channel)
	wantErr := errors.New("boom")

	err = consumer.Receive(context.Background(), func(ctx context.Context, received MessageJobPayload) error {
		_ = ctx
		_ = received
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected handler error %v, got %v", wantErr, err)
	}
	if !delivery.nacked {
		t.Fatal("expected delivery to be nacked")
	}
	if delivery.acked {
		t.Fatal("did not expect delivery to be acked")
	}
}

type fakeRabbitMQConsumerChannel struct {
	exchangeName    string
	queueName       string
	boundQueue      string
	boundRoutingKey string
	boundExchange   string
	deliveries      []*fakeRabbitMQDelivery
}

func (f *fakeRabbitMQConsumerChannel) ExchangeDeclare(name string) error {
	f.exchangeName = name
	return nil
}

func (f *fakeRabbitMQConsumerChannel) QueueDeclare(name string) error {
	f.queueName = name
	return nil
}

func (f *fakeRabbitMQConsumerChannel) QueueBind(queue string, routingKey string, exchange string) error {
	f.boundQueue = queue
	f.boundRoutingKey = routingKey
	f.boundExchange = exchange
	return nil
}

func (f *fakeRabbitMQConsumerChannel) Consume(ctx context.Context, queue string) (<-chan RabbitMQDelivery, error) {
	_ = ctx
	stream := make(chan RabbitMQDelivery, len(f.deliveries))
	for _, delivery := range f.deliveries {
		stream <- delivery
	}
	close(stream)
	return stream, nil
}

type fakeRabbitMQDelivery struct {
	body   []byte
	acked  bool
	nacked bool
}

func (f *fakeRabbitMQDelivery) Body() []byte {
	return append([]byte(nil), f.body...)
}

func (f *fakeRabbitMQDelivery) Ack() error {
	f.acked = true
	return nil
}

func (f *fakeRabbitMQDelivery) Nack(requeue bool) error {
	_ = requeue
	f.nacked = true
	return nil
}
