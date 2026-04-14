package app

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	agentcontext "DND-AI-BOT/internal/agent/context"
	agentprompt "DND-AI-BOT/internal/agent/prompt"
	agentruntime "DND-AI-BOT/internal/agent/runtime"
	"DND-AI-BOT/internal/bootstrap"
	basecontext "DND-AI-BOT/internal/context"
	"DND-AI-BOT/internal/game/rules"
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

// NewApp 完成仓库、服务、处理器和路由的装配。
func NewApp(deps *bootstrap.RuntimeDependencies) (*App, error) {
	sessionRepository := buildSessionRepository(deps)
	gameStateRepository := buildGameStateRepository(deps)
	encounterRepository := buildEncounterRepository(deps)
	sessionMemoryRepository := buildSessionMemoryRepository(deps)

	sessionMemoryService := service.NewSessionMemoryService(sessionMemoryRepository)
	gameStateService := service.NewGameStateService(gameStateRepository, sessionMemoryService)
	encounterService := service.NewEncounterService(encounterRepository)
	contextStore := basecontext.NewSessionContextStore(sessionRepository)
	contextProvider := agentcontext.NewProvider(contextStore)
	searchRuntime, err := bootstrap.BuildSearchRuntimeWithDeps(deps)
	if err != nil {
		return nil, err
	}
	agentRuntime, err := bootstrap.BuildAgentRuntime(bootstrap.AgentRuntimeInput{
		ContextProvider:  contextProvider,
		GameStateService: gameStateService,
		EncounterService: encounterService,
		RuleEngine:       rules.NewDefaultRuleEngine(nil),
		RuleSearcher:     searchRuntime.RuleSearcher,
		LoreSearcher:     searchRuntime.LoreSearcher,
	})
	if err != nil {
		return nil, err
	}
	bootstrap.LogModelAdapterReady(log.Default(), agentRuntime.Config)
	knowledgeWarmupService := service.NewKnowledgeWarmupService(
		searchRuntime.RuleSearcher,
		searchRuntime.LoreSearcher,
		gameStateRepository,
	)
	sessionSummarizer := service.NewLLMSessionSummarizer(newRuntimeModelSummaryAdapter(agentRuntime.ModelAdapter))
	sessionMemoryRefresher := service.NewSessionMemoryRefreshService(
		sessionRepository,
		sessionMemoryService,
		sessionSummarizer,
		30,
		40,
	)

	agentService := service.NewAgentService(func(ctx context.Context, input service.AgentReplyInput) (service.AgentReplyResult, error) {
		warmup, err := knowledgeWarmupService.BuildWarmup(ctx, input.SessionID)
		if err != nil {
			return service.AgentReplyResult{}, err
		}
		systemPrompt := agentprompt.ComposeSystemPrompt(input.SystemPrompt, warmup)
		memory, err := sessionMemoryService.GetBySessionID(ctx, input.SessionID)
		if err != nil {
			return service.AgentReplyResult{}, err
		}
		if memoryPrompt := agentprompt.ComposeSessionMemoryPrompt(memory); strings.TrimSpace(memoryPrompt) != "" {
			systemPrompt = strings.TrimSpace(systemPrompt + "\n\n" + memoryPrompt)
		}

		output, err := agentRuntime.Runtime.Run(ctx, agentruntime.RuntimeInput{
			SessionID:    input.SessionID,
			SystemPrompt: systemPrompt,
			UserMessage:  input.UserMessage,
			MaxSteps:     input.MaxSteps,
			ContextLimit: input.ContextLimit,
		})
		if err != nil {
			return service.AgentReplyResult{}, err
		}

		steps := make([]service.AgentStep, 0, len(output.Steps))
		for _, step := range output.Steps {
			steps = append(steps, service.AgentStep{ToolName: step.ActionName})
		}

		return service.AgentReplyResult{
			Reply: output.Reply,
			Steps: steps,
		}, nil
	}, log.Default())
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
	authHandler := httpHandler.NewAuthHandler(authService)
	authMiddleware := httpMiddleware.NewAuthMiddleware(authService)
	sessionHandler := httpHandler.NewSessionHandler(sessionService)
	gameStateHandler := httpHandler.NewGameStateHandler(gameStateService)
	encounterHandler := httpHandler.NewEncounterHandler(encounterService)

	return &App{
		Handler:                router.NewRouter(sessionHandler, gameStateHandler, encounterHandler, authHandler, authMiddleware),
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
