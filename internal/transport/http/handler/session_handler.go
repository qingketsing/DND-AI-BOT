package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/ratelimit"
	"DND-AI-BOT/internal/repository"
	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
	"DND-AI-BOT/internal/transport/http/middleware"
)

// SessionHandler 负责处理会话相关 HTTP 请求。
type SessionHandler struct {
	service    *service.SessionService
	rateLimits *ratelimit.Service
}

type successResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// NewSessionHandler 创建会话 HTTP 处理器。
type SessionHandlerOption func(*SessionHandler)

func NewSessionHandler(service *service.SessionService, options ...SessionHandlerOption) *SessionHandler {
	handler := &SessionHandler{service: service}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler
}

func WithSessionRateLimiter(rateLimits *ratelimit.Service) SessionHandlerOption {
	return func(handler *SessionHandler) {
		handler.rateLimits = rateLimits
	}
}

// CreateSession 处理创建会话请求。
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		handleServiceError(w, service.ErrUnauthorized)
		return
	}

	var request dto.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	session, err := h.service.CreateSession(r.Context(), service.CreateSessionInput{
		UserID:  user.UserID,
		Channel: model.Channel(strings.TrimSpace(request.Channel)),
	}, time.Now().UTC())
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

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		handleServiceError(w, service.ErrUnauthorized)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	session, err := h.service.GetSessionForUser(r.Context(), user.UserID, sessionID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToSessionResponse(session))
}

// ListSessions 处理当前用户的会话列表请求。
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		handleServiceError(w, service.ErrUnauthorized)
		return
	}

	sessions, err := h.service.ListSessions(r.Context(), user.UserID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToSessionListResponse(sessions))
}

// DeleteSession 处理删除会话请求。
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		handleServiceError(w, service.ErrUnauthorized)
		return
	}

	sessionID := readSessionID(r.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session id is required")
		return
	}

	if err := h.service.DeleteSession(r.Context(), user.UserID, sessionID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, successResponse{
		Success: true,
		Message: "session deleted",
	})
}

// SendMessage 处理发送消息请求。
func (h *SessionHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		handleServiceError(w, service.ErrUnauthorized)
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
	if h.rateLimits != nil {
		if err := h.rateLimits.CheckMessage(r.Context(), ratelimit.CheckInput{
			IP:        requestIP(r),
			UserID:    user.UserID,
			SessionID: sessionID,
		}); err != nil {
			handleRateLimitError(w, err)
			return
		}
	}

	session, err := h.service.SendMessage(r.Context(), user.UserID, user.DisplayName, service.SendMessageInput{
		SessionID: sessionID,
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
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, service.ErrInvalidMessage), errors.Is(err, service.ErrInvalidChannel):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, service.ErrSessionForbidden):
		writeError(w, http.StatusForbidden, "session_forbidden", err.Error())
	case errors.Is(err, repository.ErrSessionNotFound):
		writeError(w, http.StatusNotFound, "session_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

// writeJSON 统一写入 JSON 响应和状态码。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将业务错误转换成统一的 HTTP 错误响应格式。
func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, dto.NewErrorResponse(code, message))
}

// readSessionID 从 /sessions/{id} 或 /sessions/{id}/messages 路径中提取会话 ID。
func readSessionID(path string) string {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] != "sessions" {
		return ""
	}

	return parts[1]
}
