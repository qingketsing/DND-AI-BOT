package bootstrap

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"DND-AI-BOT/internal/queue"
)

const (
	defaultOutboxDispatchIntervalMS = 1000
	defaultOutboxDispatchBatchSize  = 50
)

var ErrMissingRabbitMQURL = errors.New("missing RABBITMQ_URL")

type RabbitMQConfig struct {
	URL                string
	DispatchIntervalMS int
	DispatchBatchSize  int
}

func (c RabbitMQConfig) DispatchInterval() time.Duration {
	intervalMS := c.DispatchIntervalMS
	if intervalMS <= 0 {
		intervalMS = defaultOutboxDispatchIntervalMS
	}
	return time.Duration(intervalMS) * time.Millisecond
}

func LoadRabbitMQConfigFromEnv() (RabbitMQConfig, error) {
	url := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if url == "" {
		return RabbitMQConfig{}, ErrMissingRabbitMQURL
	}
	return RabbitMQConfig{
		URL:                url,
		DispatchIntervalMS: loadIntEnv("ASYNC_MESSAGE_OUTBOX_DISPATCH_INTERVAL_MS", defaultOutboxDispatchIntervalMS),
		DispatchBatchSize:  loadIntEnv("ASYNC_MESSAGE_OUTBOX_DISPATCH_BATCH_SIZE", defaultOutboxDispatchBatchSize),
	}, nil
}

func OpenRabbitMQPublisherFromEnv() (RabbitMQConfig, queue.MessageJobPublisher, io.Closer, error) {
	config, err := LoadRabbitMQConfigFromEnv()
	if err != nil {
		return RabbitMQConfig{}, nil, nil, err
	}
	publisher, closer, err := queue.NewAMQPPublisher(config.URL)
	if err != nil {
		return RabbitMQConfig{}, nil, nil, err
	}
	return config, publisher, closer, nil
}

func parseOptionalPositiveInt(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
