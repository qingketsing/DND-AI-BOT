# Authentication Design

## Goal

Build a production-oriented authentication foundation for the Go backend and Next.js frontend that supports:

- registration with `email + password`
- login/logout and current-user lookup
- `@opencumt.org`-only registration
- cookie-based web authentication
- user ownership for chat sessions
- a clean base for later multi-session and onebot user mapping

## Scope

This spec covers:

- authentication data model
- HTTP API design
- service and repository boundaries
- cookie and session security model
- chat session ownership changes required for authenticated users

This spec does not cover:

- OAuth / third-party login
- phone-based login
- email verification
- password reset by email
- admin panel
- onebot user mapping implementation

## Product Rules

### Registration

- Users register with `email`, `password`, `confirm_password`, and `display_name`.
- Registration email must end with `@opencumt.org`.
- The email must be normalized before validation and storage:
  - trim surrounding whitespace
  - convert to lowercase
- `password` and `confirm_password` must match.
- Passwords must pass server-side strength validation.
- Successful registration automatically creates a logged-in auth session.

### Login

- Users log in with `email + password`.
- Login should return an explicit success/failure response body, while also using standard HTTP status codes.
- Invalid email and invalid password should share the same external error message to avoid account enumeration.

### Authentication State

- The browser stores only an opaque session token in an `HttpOnly` cookie.
- The backend reads the cookie and resolves the current user from Redis and PostgreSQL.
- Frontend JavaScript must not directly access the session token.

### Session Ownership

- Every business chat session belongs to a user.
- Session list, session read, and message send must all enforce user ownership.

## Architecture

The authentication system uses server-managed auth sessions instead of pure stateless JWT. PostgreSQL stores users and durable auth session rows. Redis caches auth session lookups for low-latency request authentication. The browser carries only an opaque random session token in an `HttpOnly` cookie, and the backend stores only a hash of that token.

This design is chosen over pure JWT because the project already depends on Redis, and server-side session control is a better fit for logout, disabling accounts, future multi-device management, and onebot/user binding. It keeps the frontend simple and moves authentication authority fully into the Go backend.

## Data Model

### `users`

```sql
create table users (
  id text primary key,
  email text not null unique,
  password_hash text not null,
  display_name text not null,
  status text not null,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  last_login_at timestamptz null
);
```

Rules:

- `email` is stored in normalized lowercase form.
- `status` values for the first release:
  - `active`
  - `disabled`

### `auth_sessions`

```sql
create table auth_sessions (
  id text primary key,
  user_id text not null references users(id),
  token_hash text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  last_seen_at timestamptz null,
  user_agent text null,
  ip_address text null,
  revoked_at timestamptz null
);
```

Rules:

- The browser receives the raw opaque session token.
- PostgreSQL stores only `token_hash`.
- Revoked or expired sessions are never accepted by authentication middleware.

### `sessions`

The existing business chat session model must be extended with ownership metadata.

Required fields:

```sql
alter table sessions add column user_id text references users(id);
alter table sessions add column title text null;
```

Rules:

- `user_id` is required for authenticated chat sessions.
- `title` is optional and intended for frontend multi-session display later.

## Redis Cache Model

Redis is used as an auth session cache, not as the source of truth.

Key format:

```text
auth:session:<token_hash>
```

Cached payload shape:

```json
{
  "session_id": "authsess_xxx",
  "user_id": "user_xxx",
  "expires_at": "2026-04-17T12:00:00Z",
  "revoked": false
}
```

Rules:

- Cache TTL matches session expiry.
- Redis miss falls back to PostgreSQL.
- PostgreSQL remains authoritative.

## Security Model

### Password Storage

- Passwords are hashed with `bcrypt`.
- Password hashes are never returned in any API response.

### Cookie Strategy

Cookie name:

```text
dnd_auth_session
```

Cookie attributes:

- `HttpOnly: true`
- `Path: /`
- `SameSite: Lax`
- `Secure: true` in production
- `Max-Age`: 7 days for the first release

### Why Cookie-Based Session Tokens Are Acceptable

Reading the session token from an `HttpOnly` cookie is acceptable and recommended for this web product. The security risk is not “cookie storage” by itself, but incorrect cookie and request protection settings.

Primary mitigations:

- `HttpOnly` prevents frontend JS from reading the token during XSS.
- `Secure` prevents cleartext transport in production.
- `SameSite=Lax` reduces CSRF exposure for normal browser flows.
- Sensitive auth mutations can later add `Origin` validation or a CSRF token if needed.
- The backend stores only `token_hash`, not the raw token.

### Threat Model Notes

This first version explicitly protects against:

- accidental token exposure to frontend JavaScript
- database leakage of raw session tokens
- account enumeration through login error messages

This first version does not yet include:

- email verification
- device management UI
- forced logout of all sessions
- IP reputation or rate limiting

Those can be added later without changing the core architecture.

## API Design

### `POST /auth/register`

Request:

```json
{
  "email": "user@opencumt.org",
  "password": "StrongPassword123",
  "confirm_password": "StrongPassword123",
  "display_name": "Qingke"
}
```

Success response:

```json
{
  "success": true,
  "message": "register succeeded",
  "user": {
    "id": "user_xxx",
    "email": "user@opencumt.org",
    "display_name": "Qingke"
  }
}
```

Failure examples:

```json
{
  "success": false,
  "message": "only @opencumt.org email addresses are allowed"
}
```

Possible statuses:

- `201` registration succeeded
- `400` invalid request, invalid email format, invalid email domain, password mismatch, weak password
- `409` email already registered

### `POST /auth/login`

Request:

```json
{
  "email": "user@opencumt.org",
  "password": "StrongPassword123"
}
```

Success response:

```json
{
  "success": true,
  "message": "login succeeded",
  "user": {
    "id": "user_xxx",
    "email": "user@opencumt.org",
    "display_name": "Qingke"
  }
}
```

Failure response:

```json
{
  "success": false,
  "message": "invalid email or password"
}
```

Possible statuses:

- `200` login succeeded
- `401` invalid credentials
- `403` user disabled

Rules:

- Do not reveal whether the email exists.
- Successful login refreshes the auth session cookie.

### `POST /auth/logout`

Request body:

- none

Success response:

```json
{
  "success": true,
  "message": "logout succeeded"
}
```

Behavior:

- revoke the current auth session
- delete cache entry
- clear auth cookie

### `GET /auth/me`

Success response:

```json
{
  "success": true,
  "user": {
    "id": "user_xxx",
    "email": "user@opencumt.org",
    "display_name": "Qingke"
  }
}
```

Failure:

- `401` if not authenticated

### `GET /sessions`

This is the first authenticated session-list endpoint for multi-session frontend support.

Response:

```json
{
  "items": [
    {
      "id": "session_xxx",
      "title": "法师法术准备",
      "updated_at": "2026-04-10T12:00:00Z"
    }
  ]
}
```

Rules:

- Only sessions owned by the current authenticated user are returned.

## Error Model

Service-level errors:

```go
var (
    ErrInvalidEmailFormat     = errors.New("invalid email format")
    ErrInvalidEmailDomain     = errors.New("invalid email domain")
    ErrPasswordMismatch       = errors.New("password mismatch")
    ErrWeakPassword           = errors.New("weak password")
    ErrEmailAlreadyRegistered = errors.New("email already registered")
    ErrInvalidCredentials     = errors.New("invalid credentials")
    ErrUserDisabled           = errors.New("user disabled")
    ErrAuthSessionNotFound    = errors.New("auth session not found")
    ErrUnauthorized           = errors.New("unauthorized")
)
```

Suggested HTTP error codes:

- `invalid_email_format`
- `invalid_email_domain`
- `password_mismatch`
- `weak_password`
- `email_already_registered`
- `invalid_credentials`
- `user_disabled`
- `unauthorized`

## Go Types

### `internal/model/user.go`

```go
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}
```

### `internal/model/auth_session.go`

```go
type AuthSession struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastSeenAt *time.Time
	UserAgent  *string
	IPAddress  *string
	RevokedAt  *time.Time
}
```

### `internal/model/session.go`

Required additions:

```go
type Session struct {
	ID        string
	UserID    string
	Title     string
	Channel   Channel
	History   []HistoryRecord
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

Constructor update:

```go
func NewSession(id string, userID string, channel Channel, now time.Time) *Session
```

## Repository Interfaces

### `internal/repository/user.go`

```go
type UserRepository interface {
	Save(ctx context.Context, user *model.User) error
	LoadByID(ctx context.Context, id string) (*model.User, error)
	LoadByEmail(ctx context.Context, email string) (*model.User, error)
}
```

### `internal/repository/auth_session.go`

```go
type AuthSessionRepository interface {
	Save(ctx context.Context, session *model.AuthSession) error
	LoadByTokenHash(ctx context.Context, tokenHash string) (*model.AuthSession, error)
	Revoke(ctx context.Context, sessionID string, now time.Time) error
}
```

### `internal/repository/session.go`

```go
type SessionRepository interface {
	Save(ctx context.Context, session *model.Session) error
	Load(ctx context.Context, sessionID string) (*model.Session, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.Session, error)
}
```

## Service Design

### `internal/service/auth_service.go`

```go
type AuthService struct {
	users        repository.UserRepository
	authSessions repository.AuthSessionRepository
	sessionCache AuthSessionCache
	passwords    PasswordManager
	tokenManager TokenManager
	idGenerator  IDGenerator
	clock        Clock
}
```

### Inputs and Outputs

```go
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
```

### Function Signatures

```go
func NewAuthService(
	users repository.UserRepository,
	authSessions repository.AuthSessionRepository,
	sessionCache AuthSessionCache,
	passwords PasswordManager,
	tokenManager TokenManager,
	idGenerator IDGenerator,
	clock Clock,
) *AuthService
```

```go
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthResult, error)
func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthResult, error)
func (s *AuthService) Logout(ctx context.Context, sessionToken string, now time.Time) error
func (s *AuthService) Authenticate(ctx context.Context, sessionToken string) (*model.User, *model.AuthSession, error)
func (s *AuthService) CurrentUser(ctx context.Context, sessionToken string) (*model.User, error)
```

### Validation Helpers

```go
func validateRegistrationEmail(email string) error
func isAllowedRegistrationDomain(email string) bool
func validatePassword(password string) error
```

Rules:

- `validateRegistrationEmail` must enforce `@opencumt.org` suffix after trim/lowercase normalization.
- `Register` must reject mismatched passwords before hashing.

## Support Abstractions

### `PasswordManager`

```go
type PasswordManager interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}
```

Default implementation:

- `bcrypt`

### `TokenManager`

```go
type TokenManager interface {
	NewSessionToken() (string, error)
	HashToken(token string) string
}
```

The first release uses opaque random session tokens, not JWT.

### `AuthSessionCache`

```go
type AuthSessionCache interface {
	Set(ctx context.Context, tokenHash string, session CachedAuthSession, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (CachedAuthSession, error)
	Delete(ctx context.Context, tokenHash string) error
}
```

## HTTP Middleware

### `internal/transport/http/middleware/auth.go`

```go
type AuthenticatedUser struct {
	UserID      string
	Email       string
	DisplayName string
	SessionID   string
}
```

```go
func NewAuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler
func UserFromContext(ctx context.Context) (AuthenticatedUser, bool)
```

Behavior:

1. Read `dnd_auth_session` from the request cookie.
2. Resolve the token via `AuthService.Authenticate(...)`.
3. Inject authenticated user info into request context.
4. Return `401` when authentication is required but missing/invalid.

## HTTP Handlers

### `internal/transport/http/handler/auth_handler.go`

```go
type AuthHandler struct {
	service *service.AuthService
}
```

```go
func NewAuthHandler(service *service.AuthService) *AuthHandler
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request)
```

### Cookie Helpers

```go
func writeAuthCookie(w http.ResponseWriter, token string, expiresAt time.Time)
func clearAuthCookie(w http.ResponseWriter)
func readAuthCookie(r *http.Request) (string, error)
```

## Session Service Changes

The session service must stop treating business sessions as anonymous global resources.

### `CreateSessionInput`

```go
type CreateSessionInput struct {
	UserID  string
	Channel model.Channel
}
```

### `internal/service/session_service.go`

```go
func (s *SessionService) CreateSession(ctx context.Context, input CreateSessionInput, now time.Time) (*model.Session, error)
func (s *SessionService) ListSessions(ctx context.Context, userID string) ([]*model.Session, error)
func (s *SessionService) GetSessionForUser(ctx context.Context, userID string, sessionID string) (*model.Session, error)
func (s *SessionService) SendMessage(ctx context.Context, userID string, input SendMessageInput, now time.Time) (*model.Session, error)
```

Rules:

- `userID` comes from auth middleware, not frontend request body.
- Session read and write operations must verify ownership.

## Migration and Rollout Strategy

### Phase 1

- add `users`
- add `auth_sessions`
- add auth repositories
- add auth service
- add auth handlers and cookie flow
- add auth middleware

### Phase 2

- add `user_id` and `title` to chat sessions
- enforce ownership on session APIs
- add `GET /sessions`

### Phase 3

- connect frontend login/register/logout/me flows
- move frontend session list to authenticated user scope
- prepare for onebot identity mapping

## Testing Strategy

Minimum test coverage required:

- email normalization and domain restriction
- password mismatch rejection
- duplicate email rejection
- login success and invalid credential failure
- cookie set/clear behavior in handlers
- middleware authentication success/failure
- session ownership enforcement
- `GET /sessions` returns only current user’s sessions

Include:

- service unit tests
- handler tests
- middleware tests
- repository integration tests where applicable

## Deferred Work

Explicitly deferred from this spec:

- OAuth login
- email verification links
- password reset flow
- rate limiting / brute-force lockout
- device session management UI
- JWT access token architecture

These are intentionally excluded to keep the first release focused and shippable.
