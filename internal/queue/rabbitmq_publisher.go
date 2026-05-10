package queue

import "context"

type rabbitMQPublisherChannel interface {
	ExchangeDeclare(name string) error
	QueueDeclare(name string) error
	QueueBind(queue string, routingKey string, exchange string) error
	Publish(ctx context.Context, exchange string, routingKey string, body []byte) error
}

type RabbitMQPublisher struct {
	channel rabbitMQPublisherChannel
}

func NewRabbitMQPublisher(channel rabbitMQPublisherChannel) *RabbitMQPublisher {
	return &RabbitMQPublisher{channel: channel}
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, payload MessageJobPayload) error {
	if err := p.channel.ExchangeDeclare(MessageExchange); err != nil {
		return err
	}
	if err := p.channel.QueueDeclare(MessageQueue); err != nil {
		return err
	}
	if err := p.channel.QueueBind(MessageQueue, MessageRoutingKey, MessageExchange); err != nil {
		return err
	}

	body, err := EncodeMessageJobPayload(payload)
	if err != nil {
		return err
	}
	return p.channel.Publish(ctx, MessageExchange, MessageRoutingKey, body)
}
