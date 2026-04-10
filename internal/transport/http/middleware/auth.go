package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
)

const authCookieName = "dnd_auth_session"

type contextKey string

const authenticatedUserContextKey contextKey = "authenticated_user"

type AuthenticatedUser struct {
	UserID      string
	Email       string
	DisplayName string
	SessionID   string
}

func NewAuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(authCookieName)
			if err != nil || cookie == nil || cookie.Value == "" {
				unauthorized(w)
				return
			}

			user, session, err := authService.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				unauthorized(w)
				return
			}

			ctx := withAuthenticatedUser(r.Context(), AuthenticatedUser{
				UserID:      user.ID,
				Email:       user.Email,
				DisplayName: user.DisplayName,
				SessionID:   session.ID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey).(AuthenticatedUser)
	return user, ok
}

func withAuthenticatedUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserContextKey, user)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(dto.NewErrorResponse("unauthorized", service.ErrUnauthorized.Error()))
}
