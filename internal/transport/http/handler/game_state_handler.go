package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
)

// GameStateHandler 负责处理游戏进度相关 HTTP 请求。
type GameStateHandler struct {
	service *service.GameStateService
}

// NewGameStateHandler 创建游戏进度 HTTP 处理器。
func NewGameStateHandler(service *service.GameStateService) *GameStateHandler {
	return &GameStateHandler{service: service}
}

// CreateGameState 处理创建游戏进度请求。
func (h *GameStateHandler) CreateGameState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.CreateGameStateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.Create(r.Context(), service.CreateGameStateInput{
		ID:        generateGameStateID(time.Now().UTC()),
		SessionID: sessionID,
		Player:    dto.ToPlayerState(request.Player),
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToGameStateResponse(gameState))
}

// GetGameState 处理读取游戏进度请求。
func (h *GameStateHandler) GetGameState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	gameState, err := h.service.GetBySessionID(r.Context(), sessionID)
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// UpdateStats 处理更新六维属性请求。
func (h *GameStateHandler) UpdateStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.UpdateStatsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.UpdateStats(r.Context(), service.UpdateStatsInput{
		SessionID: sessionID,
		Stats:     dto.ToCharacterStats(request.Stats),
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// AddItem 处理添加物品请求。
func (h *GameStateHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.AddItem(r.Context(), service.AddItemInput{
		SessionID: sessionID,
		Item:      dto.ToInventoryItem(request.Item),
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// RemoveItem 处理移除物品请求。
func (h *GameStateHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.RemoveItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.RemoveItem(r.Context(), service.RemoveItemInput{
		SessionID: sessionID,
		ItemID:    strings.TrimSpace(request.ItemID),
		Quantity:  request.Quantity,
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// AddGold 处理增加金币请求。
func (h *GameStateHandler) AddGold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.GoldRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.AddGold(r.Context(), service.AddGoldInput{
		SessionID: sessionID,
		Amount:    request.Amount,
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// SpendGold 处理消耗金币请求。
func (h *GameStateHandler) SpendGold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.GoldRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.SpendGold(r.Context(), service.SpendGoldInput{
		SessionID: sessionID,
		Amount:    request.Amount,
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// SetScene 处理更新当前场景请求。
func (h *GameStateHandler) SetScene(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.SetSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.SetScene(r.Context(), service.SetSceneInput{
		SessionID: sessionID,
		Scene:     request.Scene,
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

// UpsertQuest 处理新增或更新任务请求。
func (h *GameStateHandler) UpsertQuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.UpsertQuestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	gameState, err := h.service.UpsertQuest(r.Context(), service.UpsertQuestInput{
		SessionID: sessionID,
		Quest:     dto.ToQuestProgress(request.Quest),
	}, time.Now().UTC())
	if err != nil {
		handleGameStateServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToGameStateResponse(gameState))
}

func handleGameStateServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidGameState), errors.Is(err, service.ErrInsufficientGold), errors.Is(err, service.ErrInsufficientItemQuantity):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, repository.ErrGameStateNotFound):
		writeError(w, http.StatusNotFound, "game_state_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func generateGameStateID(now time.Time) string {
	return fmt.Sprintf("game-state-%d", now.UnixNano())
}
