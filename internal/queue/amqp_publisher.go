package queue

import (
	"context"
	"io"

	amqp "github.com/rabbitmq/amqp091-go"
)

type amqpPublisherChannel struct {
	channel *amqp.Channel
}

func (c *amqpPublisherChannel) ExchangeDeclare(name string) error {
	return c.channel.ExchangeDeclare(name, "direct", true, false, false, false, nil)
}

func (c *amqpPublisherChannel) QueueDeclare(name string) error {
	_, err := c.channel.QueueDeclare(name, true, false, false, false, nil)
	return err
}

func (c *amqpPublisherChannel) QueueBind(queue string, routingKey string, exchange string) error {
	return c.channel.QueueBind(queue, routingKey, exchange, false, nil)
}

func (c *amqpPublisherChannel) Publish(ctx context.Context, exchange string, routingKey string, body []byte) error {
	return c.channel.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

type amqpPublisherCloser struct {
	channel    *amqp.Channel
	connection *amqp.Connection
}

func (c *amqpPublisherCloser) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}

func NewAMQPPublisher(url string) (*RabbitMQPublisher, io.Closer, error) {
	connection, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, nil, err
	}

	return NewRabbitMQPublisher(&amqpPublisherChannel{channel: channel}), &amqpPublisherCloser{
		channel:    channel,
		connection: connection,
	}, nil
}
