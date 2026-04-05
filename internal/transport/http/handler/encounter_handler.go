package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"DND-AI-BOT/internal/game/combat"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
)

// EncounterHandler 负责处理战斗相关 HTTP 请求。
type EncounterHandler struct {
	service *service.EncounterService
}

// NewEncounterHandler 创建战斗 HTTP 处理器。
func NewEncounterHandler(service *service.EncounterService) *EncounterHandler {
	return &EncounterHandler{service: service}
}

// CreateEncounter 处理创建战斗请求。
func (h *EncounterHandler) CreateEncounter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.CreateEncounterRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	encounter, err := h.service.Create(r.Context(), service.CreateEncounterInput{
		ID:         generateEncounterID(time.Now().UTC()),
		SessionID:  sessionID,
		Combatants: dto.ToCombatants(request.Combatants),
	}, time.Now().UTC())
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToEncounterResponse(encounter))
}

// GetEncounter 处理读取战斗请求。
func (h *EncounterHandler) GetEncounter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	encounter, err := h.service.GetBySessionID(r.Context(), sessionID)
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToEncounterResponse(encounter))
}

// ApplyDamage 处理造成伤害请求。
func (h *EncounterHandler) ApplyDamage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.ApplyDamageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	encounter, err := h.service.ApplyDamage(r.Context(), service.ApplyDamageInput{
		SessionID: sessionID,
		TargetID:  strings.TrimSpace(request.TargetID),
		Amount:    request.Amount,
	}, time.Now().UTC())
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToEncounterResponse(encounter))
}

// Heal 处理治疗请求。
func (h *EncounterHandler) Heal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.HealRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	encounter, err := h.service.Heal(r.Context(), service.HealInput{
		SessionID: sessionID,
		TargetID:  strings.TrimSpace(request.TargetID),
		Amount:    request.Amount,
	}, time.Now().UTC())
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToEncounterResponse(encounter))
}

// AdvanceTurn 处理推进回合请求。
func (h *EncounterHandler) AdvanceTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	encounter, err := h.service.AdvanceTurn(r.Context(), service.AdvanceTurnInput{
		SessionID: sessionID,
	}, time.Now().UTC())
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToEncounterResponse(encounter))
}

// AddEffect 处理添加状态效果请求。
func (h *EncounterHandler) AddEffect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.AddEffectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	encounter, err := h.service.AddEffect(r.Context(), service.AddEffectInput{
		SessionID: sessionID,
		TargetID:  strings.TrimSpace(request.TargetID),
		Effect:    dto.ToStatusEffect(request.Effect),
	}, time.Now().UTC())
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToEncounterResponse(encounter))
}

// RemoveEffect 处理移除状态效果请求。
func (h *EncounterHandler) RemoveEffect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.RemoveEffectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	encounter, err := h.service.RemoveEffect(r.Context(), service.RemoveEffectInput{
		SessionID: sessionID,
		TargetID:  strings.TrimSpace(request.TargetID),
		EffectID:  strings.TrimSpace(request.EffectID),
	}, time.Now().UTC())
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToEncounterResponse(encounter))
}

// CanAct 处理检查是否可行动请求。
func (h *EncounterHandler) CanAct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.CanActRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	canAct, err := h.service.CanAct(r.Context(), service.CanActInput{
		SessionID: sessionID,
		TargetID:  strings.TrimSpace(request.TargetID),
	})
	if err != nil {
		handleEncounterServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.CanActResponse{CanAct: canAct})
}

func handleEncounterServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidEncounter), errors.Is(err, service.ErrEffectNotFound), errors.Is(err, combat.ErrInvalidDamage), errors.Is(err, combat.ErrInvalidHeal):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, repository.ErrEncounterNotFound):
		writeError(w, http.StatusNotFound, "encounter_not_found", err.Error())
	case errors.Is(err, combat.ErrCombatantNotFound):
		writeError(w, http.StatusNotFound, "combatant_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func generateEncounterID(now time.Time) string {
	return fmt.Sprintf("encounter-%d", now.UnixNano())
}
