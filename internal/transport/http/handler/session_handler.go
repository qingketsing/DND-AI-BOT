package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository/memory"
	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
)

// SessionHandler 负责处理会话相关 HTTP 请求。
type SessionHandler struct {
	service *service.SessionService
}

// NewSessionHandler 创建会话 HTTP 处理器。
func NewSessionHandler(service *service.SessionService) *SessionHandler {
	return &SessionHandler{service: service}
}

// CreateSession 处理创建会话请求。
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request dto.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	session, err := h.service.CreateSession(model.Channel(strings.TrimSpace(request.Channel)), time.Now().UTC())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToSessionResponse(session))
}

// GetSession 处理获取会话请求。
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	session, err := h.service.GetSession(sessionID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToSessionResponse(session))
}

// SendMessage 处理发送消息请求。
func (h *SessionHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	var request dto.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	session, err := h.service.SendMessage(service.SendMessageInput{
		SessionID: sessionID,
		UserID:    request.UserID,
		UserName:  request.UserName,
		Content:   request.Content,
	}, time.Now().UTC())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToSessionResponse(session))
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidMessage), errors.Is(err, service.ErrInvalidChannel), errors.Is(err, memory.ErrEmptySessionID):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, memory.ErrSessionNotFound):
		writeError(w, http.StatusNotFound, "session_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, dto.NewErrorResponse(code, message))
}

func readSessionID(path string) string {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] != "sessions" {
		return ""
	}

	return parts[1]
}
