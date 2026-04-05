package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/game/state"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
)

func TestCreateGameStateReturnsCreatedState(t *testing.T) {
	handler := newTestGameStateHandler()

	body := `{"player":{"name":"Alice","level":1,"gold":10,"stats":{"str":10,"dex":12,"con":11,"int":13,"wis":14,"cha":8},"inventory":[],"quests":[]}}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/session-1/game-state", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateGameState(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response["session_id"] != "session-1" {
		t.Fatalf("expected session_id %q, got %v", "session-1", response["session_id"])
	}
}

func TestUpdateStatsReturnsUpdatedGameState(t *testing.T) {
	handler := newTestGameStateHandlerWithState("session-1")

	request := httptest.NewRequest(http.MethodPost, "/sessions/session-1/game-state/stats", strings.NewReader(`{"stats":{"str":15,"dex":14,"con":13,"int":12,"wis":11,"cha":10}}`))
	recorder := httptest.NewRecorder()

	handler.UpdateStats(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Player struct {
			Stats struct {
				STR int `json:"str"`
				CHA int `json:"cha"`
			} `json:"stats"`
		} `json:"player"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response.Player.Stats.STR != 15 || response.Player.Stats.CHA != 10 {
		t.Fatalf("expected updated stats, got %+v", response.Player.Stats)
	}
}

func TestSpendGoldReturnsBadRequestWhenGoldIsInsufficient(t *testing.T) {
	handler := newTestGameStateHandlerWithState("session-1")

	request := httptest.NewRequest(http.MethodPost, "/sessions/session-1/game-state/gold/spend", strings.NewReader(`{"amount":99}`))
	recorder := httptest.NewRecorder()

	handler.SpendGold(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestGetGameStateReturnsNotFoundWhenMissing(t *testing.T) {
	handler := newTestGameStateHandler()

	request := httptest.NewRequest(http.MethodGet, "/sessions/missing/game-state", nil)
	recorder := httptest.NewRecorder()

	handler.GetGameState(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func newTestGameStateHandler() *GameStateHandler {
	repo := newFakeHTTPGameStateRepository()
	return NewGameStateHandler(service.NewGameStateService(repo))
}

func newTestGameStateHandlerWithState(sessionID string) *GameStateHandler {
	repo := newFakeHTTPGameStateRepository()
	now := time.Date(2026, 4, 5, 16, 0, 0, 0, time.UTC)
	repo.bySessionID[sessionID] = state.NewGameState("state-1", sessionID, state.PlayerState{
		Name:  "Alice",
		Level: 1,
		Gold:  10,
		Stats: state.CharacterStats{STR: 10, DEX: 12, CON: 11, INT: 13, WIS: 14, CHA: 8},
	}, now)
	return NewGameStateHandler(service.NewGameStateService(repo))
}

type fakeHTTPGameStateRepository struct {
	bySessionID map[string]*state.GameState
}

func newFakeHTTPGameStateRepository() *fakeHTTPGameStateRepository {
	return &fakeHTTPGameStateRepository{
		bySessionID: make(map[string]*state.GameState),
	}
}

func (f *fakeHTTPGameStateRepository) Save(ctx context.Context, gameState *state.GameState) error {
	_ = ctx
	f.bySessionID[gameState.SessionID] = gameState
	return nil
}

func (f *fakeHTTPGameStateRepository) LoadBySessionID(ctx context.Context, sessionID string) (*state.GameState, error) {
	_ = ctx
	gameState, ok := f.bySessionID[sessionID]
	if !ok {
		return nil, repository.ErrGameStateNotFound
	}
	return gameState, nil
}
