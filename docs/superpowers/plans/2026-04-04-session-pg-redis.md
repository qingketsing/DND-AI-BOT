# Session PG Redis Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist sessions to PostgreSQL and cache them in Redis through the existing repository abstraction.

**Architecture:** Use PostgreSQL as the source of truth and Redis as a cache. Keep service code depending only on `repository.SessionRepository`, implement real `postgres` and `redis` adapters, and wire them through the composite repository with write-through invalidation and cache-aside reads.

**Tech Stack:** Go, database/sql, PostgreSQL SQL, Redis client, docker compose

---

### Task 1: Session Repository Contract Shift

**Files:**
- Modify: `internal/service/session_service.go`
- Modify: `internal/service/session_service_test.go`

- [ ] **Step 1: Write the failing test**

Update `SessionService` tests so the service depends on `repository.SessionRepository` rather than the in-memory concrete type.

- [ ] **Step 2: Run test to verify it fails**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service`
Expected: FAIL with constructor or field type mismatch.

- [ ] **Step 3: Write minimal implementation**

Change `SessionService` to depend on the interface and keep behavior unchanged.

- [ ] **Step 4: Run test to verify it passes**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service`
Expected: PASS

### Task 2: PostgreSQL Session Store

**Files:**
- Create: `migrations/001_create_sessions.sql`
- Create: `migrations/002_create_session_messages.sql`
- Create: `internal/repository/postgres/session_store_impl.go`
- Create: `internal/repository/postgres/session_store_impl_test.go`

- [ ] **Step 1: Write the failing test**

Write tests for `UpsertSession` and `GetSession` using a SQL test double or a temporary database strategy that verifies session metadata and ordered messages are round-tripped.

- [ ] **Step 2: Run test to verify it fails**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/repository/postgres`
Expected: FAIL with missing `NewPGSessionStore`, `UpsertSession`, or `GetSession`.

- [ ] **Step 3: Write minimal implementation**

Implement the real store using transactions, session upsert, delete-and-reinsert message history, and ordered load.

- [ ] **Step 4: Run test to verify it passes**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/repository/postgres`
Expected: PASS

### Task 3: Redis Session Cache

**Files:**
- Create: `internal/repository/redis/session_cache_impl.go`
- Create: `internal/repository/redis/session_cache_impl_test.go`

- [ ] **Step 1: Write the failing test**

Write tests for cache `Set`, `Get`, `SetNotFound`, and `Delete` covering JSON round-trip and not-found markers.

- [ ] **Step 2: Run test to verify it fails**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/repository/redis`
Expected: FAIL with missing cache implementation symbols.

- [ ] **Step 3: Write minimal implementation**

Implement Redis key encoding, JSON serialization, not-found markers, and delete behavior.

- [ ] **Step 4: Run test to verify it passes**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/repository/redis`
Expected: PASS

### Task 4: Runtime Dependencies and App Wiring

**Files:**
- Create: `internal/bootstrap/dependencies.go`
- Modify: `internal/app/app.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Write the failing test**

Add focused tests for env-based dependency opening helpers when possible, or compile-level checks that `NewApp` accepts runtime dependencies.

- [ ] **Step 2: Run test to verify it fails**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/bootstrap ./internal/app ./cmd/api`
Expected: FAIL with missing dependency helpers or constructor mismatch.

- [ ] **Step 3: Write minimal implementation**

Open PG and Redis from env, build the real session repository, and inject it into the app.

- [ ] **Step 4: Run test to verify it passes**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/bootstrap ./internal/app ./cmd/api`
Expected: PASS

### Task 5: Full Verification

**Files:**
- Modify: `internal/service/session_service.go`
- Modify: `internal/repository/postgres/session_store_impl.go`
- Modify: `internal/repository/redis/session_cache_impl.go`
- Modify: `internal/app/app.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Run focused tests**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./internal/service ./internal/repository/composite ./internal/repository/postgres ./internal/repository/redis ./internal/bootstrap`
Expected: PASS

- [ ] **Step 2: Run full repository and app verification**

Run: `mkdir -p .cache/go-build && GOCACHE=/home/qingke/DND-AI-BOT/.cache/go-build go test ./...`
Expected: PASS except for any unrelated pre-existing failures that must be called out explicitly.
