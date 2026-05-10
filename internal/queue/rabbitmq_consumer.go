package queue

import "context"

type RabbitMQDelivery interface {
	Body() []byte
	Ack() error
	Nack(requeue bool) error
}

type rabbitMQConsumerChannel interface {
	ExchangeDeclare(name string) error
	QueueDeclare(name string) error
	QueueBind(queue string, routingKey string, exchange string) error
	Consume(ctx context.Context, queue string) (<-chan RabbitMQDelivery, error)
}

type RabbitMQConsumer struct {
	channel rabbitMQConsumerChannel
}

func NewRabbitMQConsumer(channel rabbitMQConsumerChannel) *RabbitMQConsumer {
	return &RabbitMQConsumer{channel: channel}
}

func (c *RabbitMQConsumer) Receive(ctx context.Context, handler func(context.Context, MessageJobPayload) error) error {
	if err := c.channel.ExchangeDeclare(MessageExchange); err != nil {
		return err
	}
	if err := c.channel.QueueDeclare(MessageQueue); err != nil {
		return err
	}
	if err := c.channel.QueueBind(MessageQueue, MessageRoutingKey, MessageExchange); err != nil {
		return err
	}

	deliveries, err := c.channel.Consume(ctx, MessageQueue)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case delivery, ok := <-deliveries:
		if !ok {
			return context.Canceled
		}
		payload, err := DecodeMessageJobPayload(delivery.Body())
		if err != nil {
			_ = delivery.Nack(false)
			return err
		}
		if err := handler(ctx, payload); err != nil {
			_ = delivery.Nack(true)
			return err
		}
		return delivery.Ack()
	}
}
