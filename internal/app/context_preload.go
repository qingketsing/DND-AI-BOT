package app

import (
	"context"
	"errors"

	agentcontext "DND-AI-BOT/internal/agent/context"
	agentprompt "DND-AI-BOT/internal/agent/prompt"
	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

const defaultPreloadedContextLimit = 40

type preloadedContextInput struct {
	SessionID           string
	ContextLimit        int
	ContextProvider     agentcontext.Provider
	GameStateReader     preloadedGameStateReader
	SessionMemoryReader preloadedSessionMemoryReader
}

type preloadedGameStateReader interface {
	GetBySessionID(ctx context.Context, sessionID string) (*state.GameState, error)
}

type preloadedSessionMemoryReader interface {
	GetBySessionID(ctx context.Context, sessionID string) (*model.SessionMemory, error)
}

func buildPreloadedContextPrompt(ctx context.Context, input preloadedContextInput) (string, error) {
	limit := input.ContextLimit
	if limit <= 0 {
		limit = defaultPreloadedContextLimit
	}

	var agentCtx agentcontext.AgentContext
	if input.ContextProvider != nil {
		result, err := input.ContextProvider.BuildContext(ctx, input.SessionID, limit)
		if err != nil {
			return "", err
		}
		agentCtx = result
	}

	var gameState *state.GameState
	if input.GameStateReader != nil {
		result, err := input.GameStateReader.GetBySessionID(ctx, input.SessionID)
		if err != nil && !errors.Is(err, repository.ErrGameStateNotFound) {
			return "", err
		}
		if err == nil {
			gameState = result
		}
	}

	var memory *model.SessionMemory
	if input.SessionMemoryReader != nil {
		result, err := input.SessionMemoryReader.GetBySessionID(ctx, input.SessionID)
		if err != nil {
			return "", err
		}
		memory = result
	}

	return agentprompt.ComposePreloadedSessionContextPrompt(agentCtx, gameState, memory), nil
}
