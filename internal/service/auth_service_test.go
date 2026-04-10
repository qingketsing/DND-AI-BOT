package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	rediscache "DND-AI-BOT/internal/repository/redis"
)

func TestAuthServiceRegisterRejectsInvalidDomain(t *testing.T) {
	svc := newTestAuthService()

	_, err := svc.Register(context.Background(), RegisterInput{
		Email:           "user@example.com",
		Password:        "StrongPassword123",
		ConfirmPassword: "StrongPassword123",
		DisplayName:     "Alice",
	})
	if !errors.Is(err, ErrInvalidEmailDomain) {
		t.Fatalf("expected ErrInvalidEmailDomain, got %v", err)
	}
}

func TestAuthServiceRegisterRejectsPasswordMismatch(t *testing.T) {
	svc := newTestAuthService()

	_, err := svc.Register(context.Background(), RegisterInput{
		Email:           "user@opencumt.org",
		Password:        "StrongPassword123",
		ConfirmPassword: "StrongPassword456",
		DisplayName:     "Alice",
	})
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}
}

func TestAuthServiceRegisterRejectsDuplicateEmail(t *testing.T) {
	users := newFakeUserRepository()
	users.byEmail["user@opencumt.org"] = &model.User{
		ID:           "user-existing",
		Email:        "user@opencumt.org",
		PasswordHash: "hash:StrongPassword123",
		DisplayName:  "Existing",
		Status:       model.UserStatusActive,
		CreatedAt:    time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	users.byID["user-existing"] = users.byEmail["user@opencumt.org"]

	svc := newAuthServiceForTest(users, newFakeAuthSessionRepository(), newFakeAuthSessionCache())

	_, err := svc.Register(context.Background(), RegisterInput{
		Email:           "USER@OPENCUMT.ORG",
		Password:        "StrongPassword123",
		ConfirmPassword: "StrongPassword123",
		DisplayName:     "Alice",
	})
	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}

func TestAuthServiceRegisterCreatesUserAndAuthSession(t *testing.T) {
	users := newFakeUserRepository()
	authSessions := newFakeAuthSessionRepository()
	cache := newFakeAuthSessionCache()
	svc := newAuthServiceForTest(users, authSessions, cache)

	result, err := svc.Register(context.Background(), RegisterInput{
		Email:           "  USER@OPENCUMT.ORG  ",
		Password:        "StrongPassword123",
		ConfirmPassword: "StrongPassword123",
		DisplayName:     "Alice",
		UserAgent:       "Mozilla/5.0",
		IPAddress:       "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}
	if result.SessionToken != "session-token-1" {
		t.Fatalf("expected session token to be returned, got %q", result.SessionToken)
	}
	if result.User.Email != "user@opencumt.org" {
		t.Fatalf("expected normalized email, got %q", result.User.Email)
	}
	if len(users.saved) != 1 {
		t.Fatalf("expected one user save, got %d", len(users.saved))
	}
	if users.saved[0].PasswordHash != "hash:StrongPassword123" {
		t.Fatalf("expected password hash to be stored, got %q", users.saved[0].PasswordHash)
	}
	if len(authSessions.saved) != 1 {
		t.Fatalf("expected one auth session save, got %d", len(authSessions.saved))
	}
	if authSessions.saved[0].TokenHash != "token-hash:session-token-1" {
		t.Fatalf("expected token hash to be stored, got %q", authSessions.saved[0].TokenHash)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("expected one cache set, got %d", len(cache.setCalls))
	}
}

func TestAuthServiceLoginSucceedsWithCorrectPassword(t *testing.T) {
	users := newFakeUserRepository()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-1",
		Email:        "user@opencumt.org",
		PasswordHash: "hash:StrongPassword123",
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	users.byID[user.ID] = user
	users.byEmail[user.Email] = user

	authSessions := newFakeAuthSessionRepository()
	cache := newFakeAuthSessionCache()
	svc := newAuthServiceForTest(users, authSessions, cache)

	result, err := svc.Login(context.Background(), LoginInput{
		Email:     "user@opencumt.org",
		Password:  "StrongPassword123",
		UserAgent: "Mozilla/5.0",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if result.User.ID != user.ID {
		t.Fatalf("expected logged in user %q, got %q", user.ID, result.User.ID)
	}
	if len(authSessions.saved) != 1 || len(cache.setCalls) != 1 {
		t.Fatalf("expected auth session to be persisted and cached")
	}
}

func TestAuthServiceLoginRejectsBadPassword(t *testing.T) {
	users := newFakeUserRepository()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-1",
		Email:        "user@opencumt.org",
		PasswordHash: "hash:StrongPassword123",
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	users.byID[user.ID] = user
	users.byEmail[user.Email] = user

	svc := newAuthServiceForTest(users, newFakeAuthSessionRepository(), newFakeAuthSessionCache())

	_, err := svc.Login(context.Background(), LoginInput{
		Email:    "user@opencumt.org",
		Password: "WrongPassword",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthServiceAuthenticateResolvesCacheHit(t *testing.T) {
	users := newFakeUserRepository()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-1",
		Email:        "user@opencumt.org",
		PasswordHash: "hash:StrongPassword123",
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	users.byID[user.ID] = user
	users.byEmail[user.Email] = user

	cache := newFakeAuthSessionCache()
	cache.values["token-hash:session-token-1"] = rediscache.CachedAuthSession{
		SessionID: "authsess-1",
		UserID:    user.ID,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	authSessions := newFakeAuthSessionRepository()
	svc := newAuthServiceForTest(users, authSessions, cache)

	gotUser, gotSession, err := svc.Authenticate(context.Background(), "session-token-1")
	if err != nil {
		t.Fatalf("expected authenticate to succeed, got %v", err)
	}
	if gotUser.ID != user.ID || gotSession.ID != "authsess-1" {
		t.Fatalf("expected cache hit to resolve user and session, got user=%+v session=%+v", gotUser, gotSession)
	}
	if authSessions.loadByTokenHashCalls != 0 {
		t.Fatalf("expected no auth session repository lookup on cache hit, got %d", authSessions.loadByTokenHashCalls)
	}
}

func TestAuthServiceAuthenticateFallsBackToRepositoryOnCacheMiss(t *testing.T) {
	users := newFakeUserRepository()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:           "user-1",
		Email:        "user@opencumt.org",
		PasswordHash: "hash:StrongPassword123",
		DisplayName:  "Alice",
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	users.byID[user.ID] = user
	users.byEmail[user.Email] = user

	authSessions := newFakeAuthSessionRepository()
	authSessions.byTokenHash["token-hash:session-token-1"] = &model.AuthSession{
		ID:        "authsess-1",
		UserID:    user.ID,
		TokenHash: "token-hash:session-token-1",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	cache := newFakeAuthSessionCache()
	cache.err = repository.ErrCacheMiss
	svc := newAuthServiceForTest(users, authSessions, cache)

	gotUser, gotSession, err := svc.Authenticate(context.Background(), "session-token-1")
	if err != nil {
		t.Fatalf("expected authenticate to succeed, got %v", err)
	}
	if gotUser.ID != user.ID || gotSession.ID != "authsess-1" {
		t.Fatalf("expected repository fallback to resolve user and session, got user=%+v session=%+v", gotUser, gotSession)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("expected repository fallback to repopulate cache, got %d set calls", len(cache.setCalls))
	}
}

func TestAuthServiceLogoutRevokesAndDeletesCache(t *testing.T) {
	authSessions := newFakeAuthSessionRepository()
	authSessions.byTokenHash["token-hash:session-token-1"] = &model.AuthSession{
		ID:        "authsess-1",
		UserID:    "user-1",
		TokenHash: "token-hash:session-token-1",
		ExpiresAt: time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	cache := newFakeAuthSessionCache()
	svc := newAuthServiceForTest(newFakeUserRepository(), authSessions, cache)

	now := time.Date(2026, 4, 10, 13, 0, 0, 0, time.UTC)
	if err := svc.Logout(context.Background(), "session-token-1", now); err != nil {
		t.Fatalf("expected logout to succeed, got %v", err)
	}
	if authSessions.revokedSessionID != "authsess-1" {
		t.Fatalf("expected revoke to target auth session, got %q", authSessions.revokedSessionID)
	}
	if cache.deletedTokenHash != "token-hash:session-token-1" {
		t.Fatalf("expected cache delete by token hash, got %q", cache.deletedTokenHash)
	}
}

func newTestAuthService() *AuthService {
	return newAuthServiceForTest(newFakeUserRepository(), newFakeAuthSessionRepository(), newFakeAuthSessionCache())
}

func newAuthServiceForTest(
	users *fakeUserRepository,
	authSessions *fakeAuthSessionRepository,
	cache *fakeAuthSessionCache,
) *AuthService {
	return NewAuthService(
		users,
		authSessions,
		cache,
		fakePasswordManager{},
		&fakeTokenManager{
			token: "session-token-1",
		},
		&fakeIDGenerator{},
		fakeClock{now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)},
	)
}

type fakeUserRepository struct {
	byID    map[string]*model.User
	byEmail map[string]*model.User
	saved   []*model.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byID:    make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (r *fakeUserRepository) Save(ctx context.Context, user *model.User) error {
	copyValue := *user
	r.byID[user.ID] = &copyValue
	r.byEmail[user.Email] = &copyValue
	r.saved = append(r.saved, &copyValue)
	return nil
}

func (r *fakeUserRepository) LoadByID(ctx context.Context, id string) (*model.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copyValue := *user
	return &copyValue, nil
}

func (r *fakeUserRepository) LoadByEmail(ctx context.Context, email string) (*model.User, error) {
	user, ok := r.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copyValue := *user
	return &copyValue, nil
}

type fakeAuthSessionRepository struct {
	byTokenHash          map[string]*model.AuthSession
	saved                []*model.AuthSession
	loadByTokenHashCalls int
	revokedSessionID     string
	revokedAt            time.Time
}

func newFakeAuthSessionRepository() *fakeAuthSessionRepository {
	return &fakeAuthSessionRepository{
		byTokenHash: make(map[string]*model.AuthSession),
	}
}

func (r *fakeAuthSessionRepository) Save(ctx context.Context, session *model.AuthSession) error {
	copyValue := *session
	r.byTokenHash[session.TokenHash] = &copyValue
	r.saved = append(r.saved, &copyValue)
	return nil
}

func (r *fakeAuthSessionRepository) LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error) {
	r.loadByTokenHashCalls++
	session, ok := r.byTokenHash[tokenHash]
	if !ok {
		return nil, repository.ErrAuthSessionNotFound
	}
	copyValue := *session
	return &copyValue, nil
}

func (r *fakeAuthSessionRepository) Revoke(ctx context.Context, sessionID string, now time.Time) error {
	r.revokedSessionID = sessionID
	r.revokedAt = now
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

type fakeAuthSessionCache struct {
	values            map[string]rediscache.CachedAuthSession
	err               error
	setCalls          []cacheSetCall
	deletedTokenHash  string
}

type cacheSetCall struct {
	tokenHash string
	session   rediscache.CachedAuthSession
	ttl       time.Duration
}

func newFakeAuthSessionCache() *fakeAuthSessionCache {
	return &fakeAuthSessionCache{
		values: make(map[string]rediscache.CachedAuthSession),
	}
}

func (c *fakeAuthSessionCache) Set(ctx context.Context, tokenHash string, session rediscache.CachedAuthSession, ttl time.Duration) error {
	c.values[tokenHash] = session
	c.setCalls = append(c.setCalls, cacheSetCall{
		tokenHash: tokenHash,
		session:   session,
		ttl:       ttl,
	})
	return nil
}

func (c *fakeAuthSessionCache) Get(ctx context.Context, tokenHash string) (rediscache.CachedAuthSession, error) {
	if c.err != nil {
		return rediscache.CachedAuthSession{}, c.err
	}
	session, ok := c.values[tokenHash]
	if !ok {
		return rediscache.CachedAuthSession{}, repository.ErrCacheMiss
	}
	return session, nil
}

func (c *fakeAuthSessionCache) Delete(ctx context.Context, tokenHash string) error {
	c.deletedTokenHash = tokenHash
	delete(c.values, tokenHash)
	return nil
}

type fakePasswordManager struct{}

func (fakePasswordManager) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (fakePasswordManager) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return errors.New("password mismatch")
	}
	return nil
}

type fakeTokenManager struct {
	token string
}

func (m *fakeTokenManager) NewSessionToken() (string, error) {
	return m.token, nil
}

func (m *fakeTokenManager) HashToken(token string) string {
	return "token-hash:" + token
}

type fakeClock struct {
	now time.Time
}

func (c fakeClock) Now() time.Time {
	return c.now
}

type fakeIDGenerator struct{}

func (g *fakeIDGenerator) NewID(prefix string) string {
	return prefix + "-1"
}
