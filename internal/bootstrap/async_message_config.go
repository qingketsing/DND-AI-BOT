package bootstrap

import (
	"errors"
	"os"
	"strconv"
	"time"
)

var ErrMissingAsyncMessageDependencies = errors.New("missing async message dependencies")

// AsyncMessageConfig 定义异步消息运行时配置。
type AsyncMessageConfig struct {
	Enabled                      bool
	WorkerCount                  int
	QueueBuffer                  int
	RetryDelay                   time.Duration
	RecoveryInterval             time.Duration
	RecoveryBatchSize            int
	RecoveryRetryDelay           time.Duration
	RecoveryProcessingStaleAfter time.Duration
}

// LoadAsyncMessageConfigFromEnv 从环境变量加载异步消息配置。
func LoadAsyncMessageConfigFromEnv() AsyncMessageConfig {
	return AsyncMessageConfig{
		Enabled:                      loadBoolEnv("ASYNC_MESSAGE_ENABLED", false),
		WorkerCount:                  loadIntEnv("ASYNC_MESSAGE_WORKER_COUNT", 4),
		QueueBuffer:                  loadIntEnv("ASYNC_MESSAGE_QUEUE_BUFFER", 512),
		RetryDelay:                   time.Duration(loadIntEnv("ASYNC_MESSAGE_RETRY_DELAY_MS", 200)) * time.Millisecond,
		RecoveryInterval:             time.Duration(loadIntEnv("ASYNC_MESSAGE_RECOVERY_INTERVAL_SECONDS", 30)) * time.Second,
		RecoveryBatchSize:            loadIntEnv("ASYNC_MESSAGE_RECOVERY_BATCH_SIZE", 50),
		RecoveryRetryDelay:           time.Duration(loadIntEnv("ASYNC_MESSAGE_RECOVERY_RETRY_DELAY_SECONDS", 30)) * time.Second,
		RecoveryProcessingStaleAfter: time.Duration(loadIntEnv("ASYNC_MESSAGE_PROCESSING_STALE_AFTER_SECONDS", 300)) * time.Second,
	}
}

func loadBoolEnv(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func loadIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
