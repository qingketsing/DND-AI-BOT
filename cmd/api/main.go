package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"DND-AI-BOT/internal/app"
	"DND-AI-BOT/internal/bootstrap"
	"DND-AI-BOT/internal/observability"
)

const defaultHTTPAddr = ":8080"

// main 负责初始化依赖、执行 migration 并启动 HTTP 服务。
func main() {
	logger := log.Default()
	structuredLogger := observability.DefaultLogger()
	metrics := observability.NewMetrics(nil)

	config, err := bootstrap.LoadDependencyConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := bootstrap.LogDependencyConnectivity(logger, config); err != nil {
		log.Fatal(err)
	}

	deps, err := bootstrap.OpenRuntimeDependencies()
	if err != nil {
		log.Fatal(err)
	}
	defer deps.DB.Close()
	defer deps.RedisClient.Close()

	if err := bootstrap.RunEmbeddedMigrations(context.Background(), deps.DB); err != nil {
		log.Fatal(err)
	}

	application, err := app.NewApp(deps, app.WithLogger(structuredLogger), app.WithMetrics(metrics))
	if err != nil {
		log.Fatal(err)
	}

	addr := loadHTTPAddrFromEnv()
	logger.Printf("http server listening on %s", addr)

	if err := http.ListenAndServe(addr, application.Handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func loadHTTPAddrFromEnv() string {
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}
	return defaultHTTPAddr
}
