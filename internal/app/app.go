package app

import (
	"net/http"
	"time"

	"DND-AI-BOT/internal/bootstrap"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/repository/composite"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
	"DND-AI-BOT/internal/service"
	httpHandler "DND-AI-BOT/internal/transport/http/handler"
	"DND-AI-BOT/internal/transport/http/router"
)

// App 负责承载应用初始化后的根 HTTP handler。
type App struct {
	Handler http.Handler
}

// NewApp 完成仓库、服务、处理器和路由的装配。
func NewApp(deps *bootstrap.RuntimeDependencies) *App {
	sessionRepository := buildSessionRepository(deps)
	sessionService := service.NewSessionService(sessionRepository)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)

	return &App{
		Handler: router.NewRouter(sessionHandler),
	}
}

// buildSessionRepository 根据运行时依赖组装真实的 Session 持久化仓库。
func buildSessionRepository(deps *bootstrap.RuntimeDependencies) repository.SessionRepository {
	sessionStore := postgresstore.NewPGSessionStore(deps.DB)
	sessionCache := rediscache.NewRedisSessionCache(deps.RedisClient)

	return composite.NewCompositeSessionRepository(
		sessionStore,
		sessionCache,
		composite.CachePolicy{
			BaseTTL:     10 * time.Minute,
			NotFoundTTL: 30 * time.Second,
			TTLJitter:   time.Minute,
		},
	)
}
