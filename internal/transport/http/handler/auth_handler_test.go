package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/ratelimit"
	"DND-AI-BOT/internal/repository"
	rediscache "DND-AI-BOT/internal/repository/redis"
	"DND-AI-BOT/internal/service"
)

func TestAuthHandlerRegisterSuccessSetsCookie(t *testing.T) {
	handler := newTestAuthHandler()

	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{
		"email":"user@opencumt.org",
		"password":"StrongPassword123",
		"confirm_password":"StrongPassword123",
		"display_name":"Alice"
	}`))
	recorder := httptest.NewRecorder()

	handler.Register(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	cookie := readSetCookie(t, recorder)
	if cookie.Name != authCookieName || cookie.Value == "" || !cookie.HttpOnly {
		t.Fatalf("expected auth cookie to be set, got %+v", cookie)
	}

	var response authResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if !response.Success || response.User == nil || response.User.Email != "user@opencumt.org" {
		t.Fatalf("expected successful auth response, got %+v", response)
	}
}

func TestAuthHandlerRegisterInvalidDomainReturnsBadRequest(t *testing.T) {
	handler := newTestAuthHandler()

	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{
		"email":"user@example.com",
		"password":"StrongPassword123",
		"confirm_password":"StrongPassword123",
		"display_name":"Alice"
	}`))
	recorder := httptest.NewRecorder()

	handler.Register(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	assertErrorCode(t, recorder, "invalid_email_domain")
}

func TestAuthHandlerLoginSuccessSetsCookie(t *testing.T) {
	handler := newTestAuthHandlerWithExistingUser("user@opencumt.org", "StrongPassword123")

	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{
		"email":"user@opencumt.org",
		"password":"StrongPassword123"
	}`))
	recorder := httptest.NewRecorder()

	handler.Login(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	cookie := readSetCookie(t, recorder)
	if cookie.Name != authCookieName || cookie.Value == "" {
		t.Fatalf("expected auth cookie on login, got %+v", cookie)
	}
}

func TestAuthHandlerLoginInvalidCredentialsReturnsUnauthorized(t *testing.T) {
	handler := newTestAuthHandlerWithExistingUser("user@opencumt.org", "StrongPassword123")

	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{
		"email":"user@opencumt.org",
		"password":"WrongPassword"
	}`))
	recorder := httptest.NewRecorder()

	handler.Login(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	assertErrorCode(t, recorder, "invalid_credentials")
}

func TestAuthHandlerLoginReturnsTooManyRequestsWhenRateLimited(t *testing.T) {
	handler := NewAuthHandler(
		newTestAuthService(nil),
		WithAuthRateLimiter(ratelimit.NewService(
			&fakeAuthRateLimitBackend{
				decision: ratelimit.Decision{
					Allowed:    false,
					PolicyName: "login_ip",
					RetryAfter: 30 * time.Second,
				},
			},
			ratelimit.DefaultConfig(),
			authTestClock{now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)},
		)),
	)

	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{
		"email":"user@opencumt.org",
		"password":"StrongPassword123"
	}`))
	recorder := httptest.NewRecorder()

	handler.Login(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if recorder.Header().Get("Retry-After") != "30" {
		t.Fatalf("expected Retry-After 30, got %q", recorder.Header().Get("Retry-After"))
	}
	assertErrorCode(t, recorder, "rate_limited")
}

func TestAuthHandlerLogoutClearsCookie(t *testing.T) {
	handler := newTestAuthHandlerWithExistingUser("user@opencumt.org", "StrongPassword123")

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{
		"email":"user@opencumt.org",
		"password":"StrongPassword123"
	}`))
	loginRecorder := httptest.NewRecorder()
	handler.Login(loginRecorder, loginRequest)
	loginCookie := readSetCookie(t, loginRecorder)

	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(loginCookie)
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	cookie := readSetCookie(t, recorder)
	if cookie.Name != authCookieName || cookie.MaxAge >= 0 {
		t.Fatalf("expected auth cookie to be cleared, got %+v", cookie)
	}
}

func TestAuthHandlerMeReturnsCurrentUser(t *testing.T) {
	handler := newTestAuthHandlerWithExistingUser("user@opencumt.org", "StrongPassword123")

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{
		"email":"user@opencumt.org",
		"password":"StrongPassword123"
	}`))
	loginRecorder := httptest.NewRecorder()
	handler.Login(loginRecorder, loginRequest)
	loginCookie := readSetCookie(t, loginRecorder)

	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.AddCookie(loginCookie)
	recorder := httptest.NewRecorder()

	handler.Me(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response authResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if !response.Success || response.User == nil || response.User.Email != "user@opencumt.org" {
		t.Fatalf("expected current user response, got %+v", response)
	}
}

func readSetCookie(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected response to set cookie")
	}
	return cookies[0]
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid error response, got %v", err)
	}
	if response.Error.Code != expected {
		t.Fatalf("expected error code %q, got %q", expected, response.Error.Code)
	}
}

func newTestAuthHandler() *AuthHandler {
	return NewAuthHandler(newTestAuthService(nil))
}

func newTestAuthHandlerWithExistingUser(email string, password string) *AuthHandler {
	user := &model.User{
		ID:           "user-1",
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: "hash:" + password,
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	return NewAuthHandler(newTestAuthService(user))
}

func newTestAuthService(existingUser *model.User) *service.AuthService {
	users := &authTestUserRepository{
		byID:    make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
	if existingUser != nil {
		copyValue := *existingUser
		users.byID[copyValue.ID] = &copyValue
		users.byEmail[copyValue.Email] = &copyValue
	}
	return service.NewAuthService(
		users,
		&authTestAuthSessionRepository{byTokenHash: make(map[string]*model.AuthSession)},
		&authTestAuthSessionCache{values: make(map[string]rediscache.CachedAuthSession)},
		authTestPasswordManager{},
		&authTestTokenManager{token: "session-token-1"},
		&authTestIDGenerator{},
		authTestClock{now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)},
	)
}

type authTestUserRepository struct {
	byID    map[string]*model.User
	byEmail map[string]*model.User
}

func (r *authTestUserRepository) Save(ctx context.Context, user *model.User) error {
	copyValue := *user
	r.byID[user.ID] = &copyValue
	r.byEmail[user.Email] = &copyValue
	return nil
}

func (r *authTestUserRepository) LoadByID(ctx context.Context, id string) (*model.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copyValue := *user
	return &copyValue, nil
}

func (r *authTestUserRepository) LoadByEmail(ctx context.Context, email string) (*model.User, error) {
	user, ok := r.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copyValue := *user
	return &copyValue, nil
}

type authTestAuthSessionRepository struct {
	byTokenHash map[string]*model.AuthSession
}

func (r *authTestAuthSessionRepository) Save(ctx context.Context, session *model.AuthSession) error {
	copyValue := *session
	r.byTokenHash[session.TokenHash] = &copyValue
	return nil
}

func (r *authTestAuthSessionRepository) LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error) {
	session, ok := r.byTokenHash[tokenHash]
	if !ok {
		return nil, repository.ErrAuthSessionNotFound
	}
	copyValue := *session
	return &copyValue, nil
}

func (r *authTestAuthSessionRepository) Revoke(ctx context.Context, sessionID string, now time.Time) error {
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

type authTestAuthSessionCache struct {
	values map[string]rediscache.CachedAuthSession
}

func (c *authTestAuthSessionCache) Set(ctx context.Context, tokenHash string, session rediscache.CachedAuthSession, ttl time.Duration) error {
	c.values[tokenHash] = session
	return nil
}

func (c *authTestAuthSessionCache) Get(ctx context.Context, tokenHash string) (rediscache.CachedAuthSession, error) {
	session, ok := c.values[tokenHash]
	if !ok {
		return rediscache.CachedAuthSession{}, repository.ErrCacheMiss
	}
	return session, nil
}

func (c *authTestAuthSessionCache) Delete(ctx context.Context, tokenHash string) error {
	delete(c.values, tokenHash)
	return nil
}

type authTestPasswordManager struct{}

func (authTestPasswordManager) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (authTestPasswordManager) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return errors.New("invalid password")
	}
	return nil
}

type authTestTokenManager struct {
	token string
}

func (m *authTestTokenManager) NewSessionToken() (string, error) {
	return m.token, nil
}

func (m *authTestTokenManager) HashToken(token string) string {
	return "token-hash:" + token
}

type authTestClock struct {
	now time.Time
}

func (c authTestClock) Now() time.Time {
	return c.now
}

type authTestIDGenerator struct{}

func (g *authTestIDGenerator) NewID(prefix string) string {
	return prefix + "-1"
}

type fakeAuthRateLimitBackend struct {
	decision ratelimit.Decision
}

func (f *fakeAuthRateLimitBackend) Allow(ctx context.Context, key string, policy ratelimit.Policy, now time.Time) (ratelimit.Decision, error) {
	decision := f.decision
	decision.Key = key
	decision.PolicyName = policy.Name
	decision.Limit = policy.Limit
	return decision, nil
}
