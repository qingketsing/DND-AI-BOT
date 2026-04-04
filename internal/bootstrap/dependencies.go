package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	goredis "github.com/redis/go-redis/v9"
)

var (
	// ErrMissingPostgresDSN 表示未配置数据库连接串。
	ErrMissingPostgresDSN = errors.New("missing POSTGRES_DSN")
)

// RuntimeDependencies 统一承载应用启动后可复用的外部依赖。
type RuntimeDependencies struct {
	DB          *sql.DB
	RedisClient *goredis.Client
}

// OpenPostgresFromEnv 从环境变量读取 DSN 并建立 PostgreSQL 连接。
func OpenPostgresFromEnv() (*sql.DB, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, ErrMissingPostgresDSN
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// OpenRedisFromEnv 从环境变量读取地址并建立 Redis 连接。
func OpenRedisFromEnv() (*goredis.Client, error) {
	config, err := LoadDependencyConfig()
	if err != nil {
		return nil, err
	}

	client := goredis.NewClient(&goredis.Options{
		Addr: config.RedisAddr,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// OpenRuntimeDependencies 统一初始化 PostgreSQL 和 Redis 依赖。
func OpenRuntimeDependencies() (*RuntimeDependencies, error) {
	db, err := OpenPostgresFromEnv()
	if err != nil {
		return nil, err
	}

	redisClient, err := OpenRedisFromEnv()
	if err != nil {
		db.Close()
		return nil, err
	}

	return &RuntimeDependencies{
		DB:          db,
		RedisClient: redisClient,
	}, nil
}
