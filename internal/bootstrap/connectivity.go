package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var (
	// ErrMissingPostgresAddr 表示未配置 PostgreSQL 地址。
	ErrMissingPostgresAddr = errors.New("missing POSTGRES_ADDR")
	// ErrMissingRedisAddr 表示未配置 Redis 地址。
	ErrMissingRedisAddr = errors.New("missing REDIS_ADDR")
)

// DependencyConfig 定义容器启动时依赖服务的连接地址。
type DependencyConfig struct {
	PostgresAddr string
	RedisAddr    string
}

// LoadDependencyConfig 从环境变量读取依赖服务地址。
func LoadDependencyConfig() (DependencyConfig, error) {
	config := DependencyConfig{
		PostgresAddr: os.Getenv("POSTGRES_ADDR"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
	}
	if config.PostgresAddr == "" {
		return DependencyConfig{}, ErrMissingPostgresAddr
	}
	if config.RedisAddr == "" {
		return DependencyConfig{}, ErrMissingRedisAddr
	}

	return config, nil
}

// CheckTCPConnection 校验目标地址是否可建立 TCP 连接。
func CheckTCPConnection(ctx context.Context, address string) error {
	return checkTCPConnectionWithDialer(ctx, address, defaultDialContext)
}

func checkTCPConnectionWithDialer(ctx context.Context, address string, dialContext func(context.Context, string, string) (io.Closer, error)) error {
	conn, err := dialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func defaultDialContext(ctx context.Context, network string, address string) (io.Closer, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, network, address)
}

// LogDependencyConnectivity 检查 PG 和 Redis 的网络连通性，并打印成功日志。
func LogDependencyConnectivity(logger *log.Logger, config DependencyConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := CheckTCPConnection(ctx, config.PostgresAddr); err != nil {
		return fmt.Errorf("postgres connection failed: %w", err)
	}
	logger.Printf("postgres connected: %s", config.PostgresAddr)

	if err := CheckTCPConnection(ctx, config.RedisAddr); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	logger.Printf("redis connected: %s", config.RedisAddr)

	return nil
}
