package app

import (
	"net/http"

	"../repository/memory"
	"../service"
	httpHandler "../transport/http/handler"
	"../transport/http/router"
)

// App 负责承载应用初始化后的根 HTTP handler。
type App struct {
	Handler http.Handler
}

// NewApp 完成仓库、服务、处理器和路由的装配。
func NewApp() *App {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)

	return &App{
		Handler: router.NewRouter(sessionHandler),
	}
}
