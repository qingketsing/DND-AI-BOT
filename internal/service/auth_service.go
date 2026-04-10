package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
	rediscache "DND-AI-BOT/internal/repository/redis"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmailFormat     = errors.New("invalid email format")
	ErrInvalidEmailDomain     = errors.New("invalid email domain")
	ErrPasswordMismatch       = errors.New("password mismatch")
	ErrWeakPassword           = errors.New("weak password")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrUserDisabled           = errors.New("user disabled")
	ErrUnauthorized           = errors.New("unauthorized")
)

const authSessionTTL = 7 * 24 * time.Hour

type RegisterInput struct {
	Email           string
	Password        string
	ConfirmPassword string
	DisplayName     string
	UserAgent       string
	IPAddress       string
}

type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

type AuthResult struct {
	User         model.User
	SessionToken string
	ExpiresAt    time.Time
}

type PasswordManager interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

type TokenManager interface {
	NewSessionToken() (string, error)
	HashToken(token string) string
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(prefix string) string
}

type AuthService struct {
	users        repository.UserRepository
	authSessions repository.AuthSessionRepository
	sessionCache rediscache.AuthSessionCache
	passwords    PasswordManager
	tokenManager TokenManager
	idGenerator  IDGenerator
	clock        Clock
}

func NewAuthService(
	users repository.UserRepository,
	authSessions repository.AuthSessionRepository,
	sessionCache rediscache.AuthSessionCache,
	passwords PasswordManager,
	tokenManager TokenManager,
	idGenerator IDGenerator,
	clock Clock,
) *AuthService {
	return &AuthService{
		users:        users,
		authSessions: authSessions,
		sessionCache: sessionCache,
		passwords:    passwords,
		tokenManager: tokenManager,
		idGenerator:  idGenerator,
		clock:        clock,
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	email := normalizeEmail(input.Email)
	if err := validateRegistrationEmail(email); err != nil {
		return AuthResult{}, err
	}
	if input.Password != input.ConfirmPassword {
		return AuthResult{}, ErrPasswordMismatch
	}
	if err := validatePassword(input.Password); err != nil {
		return AuthResult{}, err
	}

	existing, err := s.users.LoadByEmail(ctx, email)
	if err == nil && existing != nil {
		return AuthResult{}, ErrEmailAlreadyRegistered
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return AuthResult{}, err
	}

	passwordHash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return AuthResult{}, err
	}

	now := s.clock.Now()
	user := &model.User{
		ID:           s.idGenerator.NewID("user"),
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  strings.TrimSpace(input.DisplayName),
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Save(ctx, user); err != nil {
		return AuthResult{}, err
	}

	return s.createAuthSession(ctx, *user, input.UserAgent, input.IPAddress)
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email := normalizeEmail(input.Email)
	user, err := s.users.LoadByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}
	if user.Status == model.UserStatusDisabled {
		return AuthResult{}, ErrUserDisabled
	}
	if err := s.passwords.Compare(user.PasswordHash, input.Password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	return s.createAuthSession(ctx, *user, input.UserAgent, input.IPAddress)
}

func (s *AuthService) Logout(ctx context.Context, sessionToken string, now time.Time) error {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return ErrUnauthorized
	}

	tokenHash := s.tokenManager.HashToken(sessionToken)
	session, err := s.authSessions.LoadByTokenHash(ctx, tokenHash)
	if errors.Is(err, repository.ErrAuthSessionNotFound) {
		return ErrUnauthorized
	}
	if err != nil {
		return err
	}

	if err := s.authSessions.Revoke(ctx, session.ID, now); err != nil {
		if errors.Is(err, repository.ErrAuthSessionNotFound) {
			return ErrUnauthorized
		}
		return err
	}
	if err := s.sessionCache.Delete(ctx, tokenHash); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) Authenticate(ctx context.Context, sessionToken string) (*model.User, *model.AuthSession, error) {
	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return nil, nil, ErrUnauthorized
	}

	tokenHash := s.tokenManager.HashToken(sessionToken)
	now := s.clock.Now()

	cached, err := s.sessionCache.Get(ctx, tokenHash)
	switch {
	case err == nil:
		if cached.Revoked || !cached.ExpiresAt.After(now) {
			return nil, nil, ErrUnauthorized
		}
		user, err := s.users.LoadByID(ctx, cached.UserID)
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrUnauthorized
		}
		if err != nil {
			return nil, nil, err
		}
		if user.Status == model.UserStatusDisabled {
			return nil, nil, ErrUserDisabled
		}
		session := &model.AuthSession{
			ID:        cached.SessionID,
			UserID:    cached.UserID,
			TokenHash: tokenHash,
			ExpiresAt: cached.ExpiresAt,
		}
		return user, session, nil
	case errors.Is(err, repository.ErrCacheMiss), errors.Is(err, repository.ErrCacheNotFoundMarker):
		// Fall through to the durable store.
	default:
		return nil, nil, err
	}

	session, err := s.authSessions.LoadByTokenHash(ctx, tokenHash)
	if errors.Is(err, repository.ErrAuthSessionNotFound) {
		return nil, nil, ErrUnauthorized
	}
	if err != nil {
		return nil, nil, err
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return nil, nil, ErrUnauthorized
	}

	user, err := s.users.LoadByID(ctx, session.UserID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil, ErrUnauthorized
	}
	if err != nil {
		return nil, nil, err
	}
	if user.Status == model.UserStatusDisabled {
		return nil, nil, ErrUserDisabled
	}

	if err := s.cacheSession(ctx, tokenHash, *session, now); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *AuthService) CurrentUser(ctx context.Context, sessionToken string) (*model.User, error) {
	user, _, err := s.Authenticate(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) createAuthSession(ctx context.Context, user model.User, userAgent string, ipAddress string) (AuthResult, error) {
	now := s.clock.Now()
	expiresAt := now.Add(authSessionTTL)

	token, err := s.tokenManager.NewSessionToken()
	if err != nil {
		return AuthResult{}, err
	}
	tokenHash := s.tokenManager.HashToken(token)

	authSession := model.AuthSession{
		ID:        s.idGenerator.NewID("authsess"),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
		UserAgent: optionalString(userAgent),
		IPAddress: optionalString(ipAddress),
	}

	if err := s.authSessions.Save(ctx, &authSession); err != nil {
		return AuthResult{}, err
	}
	if err := s.cacheSession(ctx, tokenHash, authSession, now); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:         user,
		SessionToken: token,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *AuthService) cacheSession(ctx context.Context, tokenHash string, session model.AuthSession, now time.Time) error {
	ttl := session.ExpiresAt.Sub(now)
	if ttl < 0 {
		ttl = 0
	}
	return s.sessionCache.Set(ctx, tokenHash, rediscache.CachedAuthSession{
		SessionID: session.ID,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
		Revoked:   session.RevokedAt != nil,
	}, ttl)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateRegistrationEmail(email string) error {
	if email == "" {
		return ErrInvalidEmailFormat
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return ErrInvalidEmailFormat
	}
	if !isAllowedRegistrationDomain(email) {
		return ErrInvalidEmailDomain
	}
	return nil
}

func isAllowedRegistrationDomain(email string) bool {
	return strings.HasSuffix(normalizeEmail(email), "@opencumt.org")
}

func validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < 8 {
		return ErrWeakPassword
	}
	return nil
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

type BcryptPasswordManager struct{}

func (BcryptPasswordManager) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (BcryptPasswordManager) Compare(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

type SHA256TokenManager struct{}

func (SHA256TokenManager) NewSessionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (SHA256TokenManager) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type PrefixIDGenerator struct{}

func (PrefixIDGenerator) NewID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
