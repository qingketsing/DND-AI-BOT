package bootstrap

import "testing"

func TestLoadRabbitMQConfigFromEnvReadsEnvironment(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	t.Setenv("ASYNC_MESSAGE_OUTBOX_DISPATCH_INTERVAL_MS", "1500")
	t.Setenv("ASYNC_MESSAGE_OUTBOX_DISPATCH_BATCH_SIZE", "64")

	config, err := LoadRabbitMQConfigFromEnv()
	if err != nil {
		t.Fatalf("expected config load to succeed, got %v", err)
	}
	if config.URL != "amqp://guest:guest@rabbitmq:5672/" {
		t.Fatalf("expected rabbitmq url to match, got %q", config.URL)
	}
	if config.DispatchIntervalMS != 1500 {
		t.Fatalf("expected dispatch interval 1500, got %d", config.DispatchIntervalMS)
	}
	if config.DispatchBatchSize != 64 {
		t.Fatalf("expected dispatch batch size 64, got %d", config.DispatchBatchSize)
	}
}

func TestLoadRabbitMQConfigFromEnvRejectsMissingURL(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "")

	_, err := LoadRabbitMQConfigFromEnv()
	if err == nil {
		t.Fatal("expected missing rabbitmq url to fail")
	}
}
