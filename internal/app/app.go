package app

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"DND-AI-BOT/internal/agent/client"
	agentcontext "DND-AI-BOT/internal/agent/context"
	agentprompt "DND-AI-BOT/internal/agent/prompt"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/bootstrap"
	basecontext "DND-AI-BOT/internal/context"
	"DND-AI-BOT/internal/game/rules"
	"DND-AI-BOT/internal/observability"
	"DND-AI-BOT/internal/ratelimit"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/repository/composite"
	postgresstore "DND-AI-BOT/internal/repository/postgres"
	rediscache "DND-AI-BOT/internal/repository/redis"
	"DND-AI-BOT/internal/service"
	httpHandler "DND-AI-BOT/internal/transport/http/handler"
	httpMiddleware "DND-AI-BOT/internal/transport/http/middleware"
	"DND-AI-BOT/internal/transport/http/router"
)

// App 负责承载应用初始化后的根 HTTP handler。
type App struct {
	Handler                http.Handler
	AgentService           *service.AgentService
	AuthService            *service.AuthService
	SessionService         *service.SessionService
	GameStateService       *service.GameStateService
	EncounterService       *service.EncounterService
	SessionMemoryService   *service.SessionMemoryService
	SessionMemoryRefresher *service.SessionMemoryRefreshService
	KnowledgeWarmupService *service.KnowledgeWarmupService
}

type AppOptions struct {
	Logger  *slog.Logger
	Metrics *observability.Metrics
}

// AppOption 定义应用装配可选项。
type AppOption func(*AppOptions)

// WithLogger 注入结构化 logger。
func WithLogger(logger *slog.Logger) AppOption {
	return func(options *AppOptions) {
		options.Logger = logger
	}
}

// WithMetrics 注入 Prometheus metrics。
func WithMetrics(metrics *observability.Metrics) AppOption {
	return func(options *AppOptions) {
		options.Metrics = metrics
	}
}

// NewApp 完成仓库、服务、处理器和路由的装配。
func NewApp(deps *bootstrap.RuntimeDependencies, options ...AppOption) (*App, error) {
	appOptions := AppOptions{
		Logger:  observability.DefaultLogger(),
		Metrics: observability.NewNoopMetrics(),
	}
	for _, option := range options {
		if option != nil {
			option(&appOptions)
		}
	}

	securityConfig := bootstrap.LoadSecurityConfigFromEnv()
	agentObservabilityConfig := bootstrap.LoadAgentObservabilityConfigFromEnv()
	agentLatencyLogConfig := toAppAgentLatencyBreakdownLogConfig(agentObservabilityConfig.LatencyBreakdownLogConfig)
	sessionRepository := buildSessionRepository(deps)
	gameStateRepository := buildGameStateRepository(deps)
	encounterRepository := buildEncounterRepository(deps)
	sessionMemoryRepository := buildSessionMemoryRepository(deps)
	ruleEngine := rules.NewDefaultRuleEngine(nil)

	sessionMemoryService := service.NewSessionMemoryService(sessionMemoryRepository)
	gameStateService := service.NewGameStateService(gameStateRepository, sessionMemoryService)
	encounterService := service.NewEncounterService(encounterRepository, service.WithEncounterRuleEngine(ruleEngine))
	contextStore := basecontext.NewSessionContextStore(sessionRepository)
	contextProvider := agentcontext.NewProvider(contextStore)
	searchRuntime, err := bootstrap.BuildSearchRuntimeWithDeps(
		deps,
		bootstrap.WithSearchRuntimeLogger(appOptions.Logger),
		bootstrap.WithSearchRuntimeMetrics(appOptions.Metrics),
	)
	if err != nil {
		return nil, err
	}
	agentRuntime, err := bootstrap.BuildAgentRuntime(bootstrap.AgentRuntimeInput{
		ContextProvider:  contextProvider,
		GameStateService: gameStateService,
		EncounterService: encounterService,
		RuleEngine:       ruleEngine,
		RuleSearcher:     searchRuntime.RuleSearcher,
		LoreSearcher:     searchRuntime.LoreSearcher,
		Metrics:          appOptions.Metrics,
		Logger:           appOptions.Logger,
	})
	if err != nil {
		return nil, err
	}
	bootstrap.LogModelAdapterReadyForRole(log.Default(), client.ModelRolePrimary, agentRuntime.Config)
	knowledgeWarmupService := service.NewKnowledgeWarmupService(
		searchRuntime.RuleSearcher,
		searchRuntime.LoreSearcher,
		gameStateRepository,
	)
	summaryAdapter, summaryConfig, err := bootstrap.BuildModelAdapterFromEnvForRole(client.ModelRoleSummarizer)
	if err != nil {
		return nil, err
	}
	summaryAdapter = client.NewObservedModelAdapter(summaryAdapter, summaryConfig, appOptions.Metrics)
	bootstrap.LogModelAdapterReadyForRole(log.Default(), client.ModelRoleSummarizer, summaryConfig)
	sessionSummarizer := service.NewLLMSessionSummarizer(newRuntimeModelSummaryAdapter(summaryAdapter))
	sessionMemoryRefresher := service.NewSessionMemoryRefreshService(
		sessionRepository,
		sessionMemoryService,
		sessionSummarizer,
		30,
		40,
		service.WithSessionMemoryRefreshMetrics(appOptions.Metrics),
	)

	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		agentStartedAt := time.Now()
		breakdown := agentLatencyBreakdown{
			BaseSystemPromptChars: countChars(input.SystemPrompt),
			UserMessageChars:      countChars(input.UserMessage),
		}
		recordAgentPromptSegmentChars(appOptions.Metrics, "base_system_prompt", input.SystemPrompt)
		recordAgentPromptSegmentChars(appOptions.Metrics, "user_message", input.UserMessage)

		warmupStartedAt := time.Now()
		warmupResult := knowledgeWarmupService.BuildWarmupBestEffort(ctx, input.SessionID)
		breakdown.WarmupBuild = time.Since(warmupStartedAt)
		recordAgentPhase(appOptions.Metrics, "warmup_build", "success", warmupStartedAt)
		warmupText := warmupBundleText(warmupResult.Bundle)
		breakdown.WarmupBundleChars = countChars(warmupText)
		recordAgentPromptSegmentChars(appOptions.Metrics, "warmup_bundle", warmupText)

		promptStartedAt := time.Now()
		systemPrompt := agentprompt.ComposeSystemPrompt(input.SystemPrompt, warmupResult.Bundle)
		if warningsPrompt := composeWarmupWarningsPrompt(warmupResult.Warnings); strings.TrimSpace(warningsPrompt) != "" {
			systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + warningsPrompt)
		}
		breakdown.SystemPromptCompose = time.Since(promptStartedAt)
		recordAgentPhase(appOptions.Metrics, "system_prompt_compose", "success", promptStartedAt)

		preloadedStartedAt := time.Now()
		preloadedContextPrompt, err := buildPreloadedContextPrompt(ctx, preloadedContextInput{
			SessionID:           input.SessionID,
			ContextLimit:        input.ContextLimit,
			ContextProvider:     contextProvider,
			GameStateReader:     gameStateService,
			SessionMemoryReader: sessionMemoryService,
		})
		breakdown.PreloadedContextBuild = time.Since(preloadedStartedAt)
		if err != nil {
			recordAgentPhase(appOptions.Metrics, "preloaded_context_build", "error", preloadedStartedAt)
			breakdown.Total = time.Since(agentStartedAt)
			logAgentLatencyBreakdown(appOptions.Logger, input.SessionID, breakdown, agentLatencyLogConfig)
			return service.AgentReplyResult{}, err
		}
		recordAgentPhase(appOptions.Metrics, "preloaded_context_build", "success", preloadedStartedAt)
		breakdown.PreloadedContextChars = countChars(preloadedContextPrompt)
		recordAgentPromptSegmentChars(appOptions.Metrics, "preloaded_context", preloadedContextPrompt)
		if strings.TrimSpace(preloadedContextPrompt) != "" {
			systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + preloadedContextPrompt)
		}
		breakdown.FinalSystemPromptChars = countChars(systemPrompt)
		recordAgentPromptSegmentChars(appOptions.Metrics, "final_system_prompt", systemPrompt)

		runtimeStartedAt := time.Now()
		output, err := agentRuntime.Runtime.Run(ctx, agentruntime.RuntimeInput{
			SessionID:    input.SessionID,
			SystemPrompt: systemPrompt,
			UserMessage:  input.UserMessage,
			MaxSteps:     input.MaxSteps,
			ContextLimit: input.ContextLimit,
		})
		breakdown.RuntimeTotal = time.Since(runtimeStartedAt)
		if err != nil {
			recordAgentPhase(appOptions.Metrics, "runtime_total", "error", runtimeStartedAt)
			breakdown.Total = time.Since(agentStartedAt)
			logAgentLatencyBreakdown(appOptions.Logger, input.SessionID, breakdown, agentLatencyLogConfig)
			return service.AgentReplyResult{}, err
		}
		recordAgentPhase(appOptions.Metrics, "runtime_total", "success", runtimeStartedAt)
		breakdown.Total = time.Since(agentStartedAt)
		logAgentLatencyBreakdown(appOptions.Logger, input.SessionID, breakdown, agentLatencyLogConfig)

		steps := make([]service.AgentStep, 0, len(output.Steps))
		for _, step := range output.Steps {
			steps = append(steps, service.AgentStep{ToolName: step.ActionName})
		}

		return service.AgentReplyResult{
			Reply: output.Reply,
			Steps: steps,
		}, nil
	}, log.Default(), service.WithAgentLogger(appOptions.Logger), service.WithAgentMetrics(appOptions.Metrics))
	authService := service.NewAuthService(
		postgresstore.NewPGUserStore(deps.DB),
		postgresstore.NewPGAuthSessionStore(deps.DB),
		rediscache.NewRedisAuthSessionCache(deps.RedisClient),
		service.BcryptPasswordManager{},
		service.SHA256TokenManager{},
		service.PrefixIDGenerator{},
		service.SystemClock{},
	)
	sessionService := service.NewSessionService(sessionRepository, agentService)
	sessionService.SetMemoryRefresher(sessionMemoryRefresher)
	sessionDeleteCleaners := make([]service.SessionDeleteCleaner, 0, 3)
	if cleaner, ok := gameStateRepository.(service.SessionDeleteCleaner); ok {
		sessionDeleteCleaners = append(sessionDeleteCleaners, cleaner)
	}
	if cleaner, ok := encounterRepository.(service.SessionDeleteCleaner); ok {
		sessionDeleteCleaners = append(sessionDeleteCleaners, cleaner)
	}
	if cleaner, ok := sessionMemoryRepository.(service.SessionDeleteCleaner); ok {
		sessionDeleteCleaners = append(sessionDeleteCleaners, cleaner)
	}
	sessionService.SetDeleteCleaners(sessionDeleteCleaners...)
	rateLimitService := buildRateLimitService(deps, appOptions.Metrics)
	authHandler := httpHandler.NewAuthHandler(
		authService,
		httpHandler.WithAuthRateLimiter(rateLimitService),
		httpHandler.WithCookieConfig(toHandlerCookieConfig(securityConfig.Cookie)),
	)
	authMiddleware := httpMiddleware.NewAuthMiddleware(authService)
	sessionHandler := httpHandler.NewSessionHandler(sessionService, httpHandler.WithSessionRateLimiter(rateLimitService))
	gameStateHandler := httpHandler.NewGameStateHandler(gameStateService)
	encounterHandler := httpHandler.NewEncounterHandler(encounterService)
	metricsHandler := httpMiddleware.NewMetricsAccessMiddleware(toMiddlewareMetricsAccessConfig(securityConfig.Metrics))(appOptions.Metrics.Handler())

	return &App{
		Handler: router.NewRouter(
			sessionHandler,
			gameStateHandler,
			encounterHandler,
			authHandler,
			authMiddleware,
			router.WithMetricsHandler(metricsHandler),
			router.WithGlobalMiddleware(
				httpMiddleware.NewSecurityMiddleware(toMiddlewareSecurityConfig(securityConfig)),
				httpMiddleware.NewRequestIDMiddleware(),
				httpMiddleware.NewMetricsMiddleware(appOptions.Metrics),
				httpMiddleware.NewAccessLogMiddleware(appOptions.Logger),
			),
		),
		AgentService:           agentService,
		AuthService:            authService,
		SessionService:         sessionService,
		GameStateService:       gameStateService,
		EncounterService:       encounterService,
		SessionMemoryService:   sessionMemoryService,
		SessionMemoryRefresher: sessionMemoryRefresher,
		KnowledgeWarmupService: knowledgeWarmupService,
	}, nil
}

func toAppAgentLatencyBreakdownLogConfig(config bootstrap.AgentLatencyBreakdownLogConfig) AgentLatencyBreakdownLogConfig {
	return NormalizeAgentLatencyBreakdownLogConfig(AgentLatencyBreakdownLogConfig{
		Mode:      AgentLatencyBreakdownLogMode(config.Mode),
		Threshold: config.Threshold,
	})
}

func toMiddlewareSecurityConfig(config bootstrap.SecurityConfig) httpMiddleware.SecurityConfig {
	return httpMiddleware.SecurityConfig{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   config.AllowedMethods,
		AllowedHeaders:   config.AllowedHeaders,
		AllowCredentials: config.AllowCredentials,
		MaxBodyBytes:     config.MaxBodyBytes,
		TrustedProxies:   config.TrustedProxies,
	}
}

func toMiddlewareMetricsAccessConfig(config bootstrap.MetricsAccessConfig) httpMiddleware.MetricsAccessConfig {
	return httpMiddleware.MetricsAccessConfig{
		Enabled:      config.Enabled,
		AllowedCIDRs: config.AllowedCIDRs,
		BearerToken:  config.BearerToken,
	}
}

func toHandlerCookieConfig(config bootstrap.CookieConfig) httpHandler.CookieConfig {
	return httpHandler.CookieConfig{
		Secure:   config.Secure,
		SameSite: config.SameSite,
		Domain:   config.Domain,
	}
}

func buildRateLimitService(deps *bootstrap.RuntimeDependencies, metrics *observability.Metrics) *ratelimit.Service {
	if deps == nil || deps.RedisClient == nil {
		return nil
	}
	return ratelimit.NewService(
		ratelimit.NewRedisLimiter(deps.RedisClient, "dnd:ratelimit"),
		ratelimit.LoadConfigFromEnv(),
		service.SystemClock{},
		ratelimit.WithMetrics(metrics),
	)
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

// buildGameStateRepository 根据运行时依赖组装真实的 GameState 持久化仓库。
func buildGameStateRepository(deps *bootstrap.RuntimeDependencies) repository.GameStateRepository {
	gameStateStore := postgresstore.NewPGGameStateStore(deps.DB)
	gameStateCache := rediscache.NewRedisGameStateCache(deps.RedisClient)

	return composite.NewCompositeGameStateRepository(
		gameStateStore,
		gameStateCache,
		composite.CachePolicy{
			BaseTTL:     10 * time.Minute,
			NotFoundTTL: 30 * time.Second,
			TTLJitter:   time.Minute,
		},
	)
}

// buildEncounterRepository 根据运行时依赖组装真实的 Encounter 持久化仓库。
func buildEncounterRepository(deps *bootstrap.RuntimeDependencies) repository.EncounterRepository {
	encounterStore := postgresstore.NewPGEncounterStore(deps.DB)
	encounterCache := rediscache.NewRedisEncounterCache(deps.RedisClient)

	return composite.NewCompositeEncounterRepository(
		encounterStore,
		encounterCache,
		composite.CachePolicy{
			BaseTTL:     10 * time.Minute,
			NotFoundTTL: 30 * time.Second,
			TTLJitter:   time.Minute,
		},
	)
}

// buildSessionMemoryRepository 根据运行时依赖组装真实的 SessionMemory 持久化仓库。
func buildSessionMemoryRepository(deps *bootstrap.RuntimeDependencies) repository.SessionMemoryRepository {
	sessionMemoryStore := postgresstore.NewPGSessionMemoryStore(deps.DB)
	sessionMemoryCache := rediscache.NewRedisSessionMemoryCache(deps.RedisClient)

	return composite.NewCompositeSessionMemoryRepository(
		sessionMemoryStore,
		sessionMemoryCache,
		composite.CachePolicy{
			BaseTTL:     10 * time.Minute,
			NotFoundTTL: 30 * time.Second,
			TTLJitter:   time.Minute,
		},
	)
}
