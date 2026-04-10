package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
)

const authCookieName = "dnd_auth_session"

type AuthHandler struct {
	service *service.AuthService
}

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	DisplayName     string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type authResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	User    *authUserResponse `json:"user,omitempty"`
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	result, err := h.service.Register(r.Context(), service.RegisterInput{
		Email:           request.Email,
		Password:        request.Password,
		ConfirmPassword: request.ConfirmPassword,
		DisplayName:     request.DisplayName,
		UserAgent:       r.UserAgent(),
		IPAddress:       requestIP(r),
	})
	if err != nil {
		handleAuthServiceError(w, err)
		return
	}

	writeAuthCookie(w, result.SessionToken, result.ExpiresAt)
	writeJSON(w, http.StatusCreated, authResponse{
		Success: true,
		Message: "register succeeded",
		User:    toAuthUserResponse(result.User),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	result, err := h.service.Login(r.Context(), service.LoginInput{
		Email:     request.Email,
		Password:  request.Password,
		UserAgent: r.UserAgent(),
		IPAddress: requestIP(r),
	})
	if err != nil {
		handleAuthServiceError(w, err)
		return
	}

	writeAuthCookie(w, result.SessionToken, result.ExpiresAt)
	writeJSON(w, http.StatusOK, authResponse{
		Success: true,
		Message: "login succeeded",
		User:    toAuthUserResponse(result.User),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, err := readAuthCookie(r)
	if err != nil {
		handleAuthServiceError(w, service.ErrUnauthorized)
		return
	}

	if err := h.service.Logout(r.Context(), token, time.Now().UTC()); err != nil {
		handleAuthServiceError(w, err)
		return
	}

	clearAuthCookie(w)
	writeJSON(w, http.StatusOK, authResponse{
		Success: true,
		Message: "logout succeeded",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, err := readAuthCookie(r)
	if err != nil {
		handleAuthServiceError(w, service.ErrUnauthorized)
		return
	}

	user, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		handleAuthServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Success: true,
		Message: "current user loaded",
		User:    toAuthUserResponse(*user),
	})
}

func writeAuthCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		Expires:  expiresAt,
		MaxAge:   max(0, int(time.Until(expiresAt).Seconds())),
	})
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
}

func readAuthCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", http.ErrNoCookie
	}
	return token, nil
}

func handleAuthServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidEmailFormat):
		writeError(w, http.StatusBadRequest, "invalid_email_format", err.Error())
	case errors.Is(err, service.ErrInvalidEmailDomain):
		writeError(w, http.StatusBadRequest, "invalid_email_domain", err.Error())
	case errors.Is(err, service.ErrPasswordMismatch):
		writeError(w, http.StatusBadRequest, "password_mismatch", err.Error())
	case errors.Is(err, service.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		writeError(w, http.StatusConflict, "email_already_registered", err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, service.ErrUserDisabled):
		writeError(w, http.StatusForbidden, "user_disabled", err.Error())
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	default:
		writeJSON(w, http.StatusInternalServerError, dto.NewErrorResponse("internal_error", "internal server error"))
	}
}

func toAuthUserResponse(user model.User) *authUserResponse {
	return &authUserResponse{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
}

func requestIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
