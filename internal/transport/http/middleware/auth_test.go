package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	rediscache "DND-AI-BOT/internal/repository/redis"
	"DND-AI-BOT/internal/service"
)

func TestAuthMiddlewareMissingCookieReturnsUnauthorized(t *testing.T) {
	middleware := NewAuthMiddleware(newMiddlewareTestAuthService())

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected protected handler not to run without cookie")
	}))

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAuthMiddlewareInvalidCookieReturnsUnauthorized(t *testing.T) {
	middleware := NewAuthMiddleware(newMiddlewareTestAuthService())

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected protected handler not to run with invalid cookie")
	}))

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "missing-token"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAuthMiddlewareInjectsAuthenticatedUserIntoContext(t *testing.T) {
	authService := newMiddlewareTestAuthService()
	_, err := authService.Register(context.Background(), service.RegisterInput{
		Email:           "user@opencumt.org",
		Password:        "StrongPassword123",
		ConfirmPassword: "StrongPassword123",
		DisplayName:     "Alice",
	})
	if err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	middleware := NewAuthMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected authenticated user in context")
		}
		if user.Email != "user@opencumt.org" || user.DisplayName != "Alice" || user.UserID == "" || user.SessionID == "" {
			t.Fatalf("unexpected authenticated user: %+v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token-1"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func newMiddlewareTestAuthService() *service.AuthService {
	return service.NewAuthService(
		&middlewareTestUserRepository{
			byID:    make(map[string]*model.User),
			byEmail: make(map[string]*model.User),
		},
		&middlewareTestAuthSessionRepository{
			byTokenHash: make(map[string]*model.AuthSession),
		},
		&middlewareTestAuthSessionCache{
			values: make(map[string]rediscache.CachedAuthSession),
		},
		middlewareTestPasswordManager{},
		&middlewareTestTokenManager{token: "session-token-1"},
		&middlewareTestIDGenerator{},
		middlewareTestClock{now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)},
	)
}

type middlewareTestUserRepository struct {
	byID    map[string]*model.User
	byEmail map[string]*model.User
}

func (r *middlewareTestUserRepository) Save(ctx context.Context, user *model.User) error {
	copyValue := *user
	r.byID[user.ID] = &copyValue
	r.byEmail[user.Email] = &copyValue
	return nil
}

func (r *middlewareTestUserRepository) LoadByID(ctx context.Context, id string) (*model.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copyValue := *user
	return &copyValue, nil
}

func (r *middlewareTestUserRepository) LoadByEmail(ctx context.Context, email string) (*model.User, error) {
	user, ok := r.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copyValue := *user
	return &copyValue, nil
}

type middlewareTestAuthSessionRepository struct {
	byTokenHash map[string]*model.AuthSession
}

func (r *middlewareTestAuthSessionRepository) Save(ctx context.Context, session *model.AuthSession) error {
	copyValue := *session
	r.byTokenHash[session.TokenHash] = &copyValue
	return nil
}

func (r *middlewareTestAuthSessionRepository) LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error) {
	session, ok := r.byTokenHash[tokenHash]
	if !ok {
		return nil, repository.ErrAuthSessionNotFound
	}
	copyValue := *session
	return &copyValue, nil
}

func (r *middlewareTestAuthSessionRepository) Revoke(ctx context.Context, sessionID string, now time.Time) error {
	for tokenHash, session := range r.byTokenHash {
		if session.ID == sessionID {
			copyValue := *session
			copyValue.RevokedAt = &now
			copyValue.UpdatedAt = now
			r.byTokenHash[tokenHash] = &copyValue
			return nil
		}
	}
	return repository.ErrAuthSessionNotFound
}

type middlewareTestAuthSessionCache struct {
	values map[string]rediscache.CachedAuthSession
}

func (c *middlewareTestAuthSessionCache) Set(ctx context.Context, tokenHash string, session rediscache.CachedAuthSession, ttl time.Duration) error {
	c.values[tokenHash] = session
	return nil
}

func (c *middlewareTestAuthSessionCache) Get(ctx context.Context, tokenHash string) (rediscache.CachedAuthSession, error) {
	session, ok := c.values[tokenHash]
	if !ok {
		return rediscache.CachedAuthSession{}, repository.ErrCacheMiss
	}
	return session, nil
}

func (c *middlewareTestAuthSessionCache) Delete(ctx context.Context, tokenHash string) error {
	delete(c.values, tokenHash)
	return nil
}

type middlewareTestPasswordManager struct{}

func (middlewareTestPasswordManager) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (middlewareTestPasswordManager) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return errors.New("invalid password")
	}
	return nil
}

type middlewareTestTokenManager struct {
	token string
}

func (m *middlewareTestTokenManager) NewSessionToken() (string, error) {
	return m.token, nil
}

func (m *middlewareTestTokenManager) HashToken(token string) string {
	return "token-hash:" + token
}

type middlewareTestClock struct {
	now time.Time
}

func (c middlewareTestClock) Now() time.Time {
	return c.now
}

type middlewareTestIDGenerator struct{}

func (g *middlewareTestIDGenerator) NewID(prefix string) string {
	return prefix + "-1"
}
