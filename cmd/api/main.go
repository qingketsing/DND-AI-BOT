package main

import (
	"log"

	"DND-AI-BOT/internal/app"
	"DND-AI-BOT/internal/bootstrap"
)

// main 负责初始化运行时依赖，并在容器内保持进程存活以便观察连接状态。
func main() {
	logger := log.Default()

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

	application, err := app.NewApp(deps)
	if err != nil {
		log.Fatal(err)
	}
	_ = application

	logger.Print("dependencies connected and repositories initialized, container is running")
	select {}
}
