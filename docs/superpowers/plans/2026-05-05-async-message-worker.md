# 异步 Worker 化消息处理实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将消息发送链路从同步阻塞 HTTP 改造成“入队即返回、后台 Worker 异步处理、前端轮询结果”的可扩展架构，支撑 `100+` 并发消息请求进入系统。

**Architecture:** 网关层负责鉴权、落库、创建 `message_job` 和发布 RabbitMQ 任务；Worker 池消费任务并通过 Redis `session lock` 保证同一会话串行执行；PostgreSQL 作为状态真相保存 user message、assistant reply 和 job 状态；前端通过 `GET /messages/{id}` 轮询任务状态。实现顺序采用最小闭环优先：先打通异步入队和状态查询，再接真实 Agent 执行，最后补重试、续约和监控。

**Tech Stack:** Go, PostgreSQL, RabbitMQ, Redis, 现有 Agent Runtime, 现有 HTTP Handler / DTO / Service / Repository 模式

---

### Task 1: 任务表与 Repository 闭环

**Files:**
- Create: `internal/model/message_job.go`
- Create: `internal/repository/postgres/message_job_repository.go`
- Create: `internal/repository/postgres/message_job_repository_test.go`
- Modify: `internal/repository/postgres/migrations/...` 或现有迁移目录下新增 `message_jobs` migration
- Modify: `internal/repository/postgres/repository.go` 或现有仓储装配文件

- [ ] **Step 1: 写 `message_jobs` migration 的失败验证**

验证目标：
- 能创建 `message_jobs` 表
- 包含 `queued / processing / completed / retryable_failed / failed / cancelled` 所需字段
- 建立 `status`、`session_id`、`message_id` 索引

建议 migration 核心 SQL：

```sql
CREATE TABLE message_jobs (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    worker_id TEXT NOT NULL DEFAULT '',
    queued_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    last_error_code TEXT NOT NULL DEFAULT '',
    last_error_message TEXT NOT NULL DEFAULT '',
    latency_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_message_jobs_status_created_at
    ON message_jobs(status, created_at);
CREATE INDEX idx_message_jobs_session_id_created_at
    ON message_jobs(session_id, created_at);
CREATE INDEX idx_message_jobs_message_id
    ON message_jobs(message_id);
CREATE INDEX idx_message_jobs_user_id_created_at
    ON message_jobs(user_id, created_at);
```

- [ ] **Step 2: 运行 repository/migration 测试并确认失败**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/repository/postgres -run 'TestMessageJobRepository|TestMigrations'
```

Expected:
- 因 `message_jobs` 不存在或仓储未实现而失败

- [ ] **Step 3: 实现 `MessageJob` 模型和 Repository 接口**

核心类型：

```go
type MessageJobStatus string

const (
    MessageJobQueued          MessageJobStatus = "queued"
    MessageJobProcessing      MessageJobStatus = "processing"
    MessageJobCompleted       MessageJobStatus = "completed"
    MessageJobRetryableFailed MessageJobStatus = "retryable_failed"
    MessageJobFailed          MessageJobStatus = "failed"
    MessageJobCancelled       MessageJobStatus = "cancelled"
)

type MessageJob struct {
    ID               string
    MessageID        string
    SessionID        string
    UserID           string
    Status           MessageJobStatus
    AttemptCount     int
    MaxAttempts      int
    WorkerID         string
    QueuedAt         time.Time
    StartedAt        *time.Time
    FinishedAt       *time.Time
    LastErrorCode    string
    LastErrorMessage string
    LatencyMS        int64
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

Repository 最小接口：

```go
type MessageJobRepository interface {
    Create(ctx context.Context, job model.MessageJob) error
    GetByID(ctx context.Context, jobID string) (*model.MessageJob, error)
    GetByMessageID(ctx context.Context, messageID string) (*model.MessageJob, error)
    MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error
    MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error
    MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error
    MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error
    IncrementAttempt(ctx context.Context, jobID string) error
}
```

- [ ] **Step 4: 重新运行 Repository 测试并确认通过**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/repository/postgres -run 'TestMessageJobRepository|TestMigrations'
```

Expected:
- PASS

- [ ] **Step 5: 提交**

```bash
git add internal/model/message_job.go internal/repository/postgres/message_job_repository.go internal/repository/postgres/message_job_repository_test.go
git add internal/repository/postgres/migrations
git commit -m "update: Add message job persistence model"
```

### Task 2: RabbitMQ Job 发布/消费与 Redis Session Lock

**Files:**
- Create: `internal/queue/message_job_payload.go`
- Create: `internal/queue/message_job_publisher.go`
- Create: `internal/queue/message_job_consumer.go`
- Create: `internal/queue/message_job_publisher_test.go`
- Create: `internal/queue/message_job_consumer_test.go`
- Create: `internal/worker/session_lock.go`
- Create: `internal/worker/session_lock_test.go`

- [ ] **Step 1: 写失败测试，覆盖 payload 序列化、publisher、consumer、session lock**

覆盖：
- `MessageJobPayload` JSON 编解码
- 发布时 routing key 为 `message.process`
- consumer 能把 MQ body 还原为 payload
- Redis `SET NX EX` 抢锁成功/失败
- compare-and-delete 解锁

核心 payload：

```go
type MessageJobPayload struct {
    JobID     string    `json:"job_id"`
    MessageID string    `json:"message_id"`
    SessionID string    `json:"session_id"`
    UserID    string    `json:"user_id"`
    Attempt   int       `json:"attempt"`
    QueuedAt  time.Time `json:"queued_at"`
}
```

- [ ] **Step 2: 运行 queue/worker 测试并确认失败**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/queue ./internal/worker -run 'TestMessageJob|TestSessionLock'
```

Expected:
- 因 publisher/consumer/lock 未实现而失败

- [ ] **Step 3: 实现 RabbitMQ 发布与消费接口**

最小接口：

```go
type MessageJobPublisher interface {
    Publish(ctx context.Context, payload MessageJobPayload) error
}

type MessageJobConsumer interface {
    Start(ctx context.Context, handler func(context.Context, MessageJobPayload) error) error
}
```

exchange / queue 约定：

```go
const (
    MessageExchange   = "agent.message"
    MessageQueue      = "agent.message.default"
    MessageRoutingKey = "message.process"
)
```

- [ ] **Step 4: 实现 Redis `session lock`**

最小接口：

```go
type SessionLock interface {
    Acquire(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) (bool, error)
    Renew(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) error
    Release(ctx context.Context, sessionID string, jobID string, workerID string) error
}
```

键格式：

```text
session:{session_id}:processing_lock
```

- [ ] **Step 5: 重新运行 queue/worker 测试并确认通过**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/queue ./internal/worker -run 'TestMessageJob|TestSessionLock'
```

Expected:
- PASS

- [ ] **Step 6: 提交**

```bash
git add internal/queue internal/worker
git commit -m "update: Add async message queue and session lock primitives"
```

### Task 3: 异步入队 Service 与 HTTP API

**Files:**
- Create: `internal/service/async_message_service.go`
- Create: `internal/service/async_message_service_test.go`
- Create: `internal/transport/http/dto/async_message_dto.go`
- Modify: `internal/transport/http/handler/session_handler.go`
- Modify: `internal/transport/http/handler/session_handler_test.go`
- Modify: `internal/transport/http/router/...` 或现有路由装配文件

- [ ] **Step 1: 写失败测试，覆盖异步入队和消息状态查询**

覆盖：
- `POST /sessions/{id}/messages` 返回 `202`
- 响应体包含 `message_id / job_id / status=queued`
- `GET /messages/{id}` 返回 `queued / processing / completed / failed`
- 会话归属校验失败返回 `403`

异步 service 输入输出：

```go
type EnqueueMessageInput struct {
    SessionID string
    UserID    string
    Content   string
}

type EnqueueMessageResult struct {
    MessageID string
    JobID     string
    SessionID string
    Status    string
}
```

- [ ] **Step 2: 运行 service/handler 测试并确认失败**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/service ./internal/transport/http/handler -run 'TestAsyncMessage|TestSessionHandler'
```

Expected:
- 因异步 service 和新 handler 路由不存在而失败

- [ ] **Step 3: 实现 `AsyncMessageService`**

职责：
- 校验 session 归属
- 写 user message
- 创建 `message_job`
- 发布 MQ job
- 返回 `queued`

最小接口：

```go
type AsyncMessageService interface {
    EnqueueMessage(ctx context.Context, input EnqueueMessageInput) (*EnqueueMessageResult, error)
    GetMessageStatus(ctx context.Context, messageID string, userID string) (*MessageStatusResult, error)
}
```

- [ ] **Step 4: 修改 HTTP handler，使 `POST /messages` 返回 `202 Accepted`**

响应示例：

```json
{
  "message_id": "msg_xxx",
  "job_id": "job_xxx",
  "session_id": "session_xxx",
  "status": "queued"
}
```

新增：

```http
GET /messages/{message_id}
```

- [ ] **Step 5: 重新运行 service/handler 测试并确认通过**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/service ./internal/transport/http/handler -run 'TestAsyncMessage|TestSessionHandler'
```

Expected:
- PASS

- [ ] **Step 6: 提交**

```bash
git add internal/service/async_message_service.go internal/service/async_message_service_test.go
git add internal/transport/http/dto/async_message_dto.go
git add internal/transport/http/handler/session_handler.go internal/transport/http/handler/session_handler_test.go
git add internal/transport/http/router
git commit -m "update: Add async message enqueue API"
```

### Task 4: Worker Processor 接入真实 Agent Runner

**Files:**
- Create: `internal/worker/agent_message_job_processor.go`
- Create: `internal/worker/agent_message_job_processor_test.go`
- Modify: `internal/service/agent_service.go` 或复用现有 `AgentRunner` 抽象
- Modify: `internal/model/message.go` 或消息写入相关 repository

- [ ] **Step 1: 写失败测试，覆盖 worker 成功路径和失败路径**

覆盖：
- 成功处理时写入 assistant message 并将 job 标记 `completed`
- `ErrSessionBusy` 时不写 assistant message，返回可重试错误
- 临时 Agent 错误时标记 `retryable_failed`
- 不可恢复错误时标记 `failed`
- 同一 job 重复执行时不重复写 assistant message

最小处理器：

```go
type WorkerProcessor interface {
    ProcessMessageJob(ctx context.Context, payload queue.MessageJobPayload) error
}
```

- [ ] **Step 2: 运行 worker processor 测试并确认失败**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/worker -run 'TestAgentMessageJobProcessor'
```

Expected:
- 因处理器未实现而失败

- [ ] **Step 3: 实现 `AgentMessageJobProcessor`**

顺序必须是：

1. 读取 job
2. 已完成则直接返回
3. 抢 session lock
4. 标记 job 为 `processing`
5. 运行 agent runner
6. 写 assistant message
7. 标记 `completed`
8. 释放锁

建议结构：

```go
type AgentMessageJobProcessor struct {
    jobs        MessageJobRepository
    messages    MessageRepository
    sessions    SessionRepository
    lock        SessionLock
    agentRunner AgentRunner
    logger      *slog.Logger
}
```

- [ ] **Step 4: 重新运行 worker processor 测试并确认通过**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/worker -run 'TestAgentMessageJobProcessor'
```

Expected:
- PASS

- [ ] **Step 5: 提交**

```bash
git add internal/worker/agent_message_job_processor.go internal/worker/agent_message_job_processor_test.go
git add internal/service/agent_service.go internal/model/message.go
git commit -m "update: Add async agent message worker processor"
```

### Task 5: Bootstrap、配置切换与后台 Worker 启动

**Files:**
- Modify: `internal/bootstrap/...` 中 RabbitMQ / Redis / app 组装文件
- Modify: `internal/app/app.go`
- Modify: `cmd/...` 启动入口或现有 server main
- Create: `internal/worker/runner.go`
- Create: `internal/worker/runner_test.go`

- [ ] **Step 1: 写失败测试，覆盖 async 模式装配和 worker 启动**

覆盖：
- `MESSAGE_EXECUTION_MODE=async` 时装配 `AsyncMessageService`
- worker runner 能启动 consumer 并调用 processor
- `sync` 模式保留现有同步链路

建议配置：

```go
type MessageExecutionMode string

const (
    MessageExecutionModeSync  MessageExecutionMode = "sync"
    MessageExecutionModeAsync MessageExecutionMode = "async"
)
```

- [ ] **Step 2: 运行 bootstrap/app 测试并确认失败**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/bootstrap ./internal/app ./internal/worker -run 'Test.*Async|TestWorkerRunner'
```

Expected:
- 因 async 装配和 worker runner 未接入而失败

- [ ] **Step 3: 接入配置开关和 Worker Runner**

要求：
- 默认仍可保留 `sync`
- `async` 模式下启动 MQ consumer + processor
- Gateway handler 使用 `AsyncMessageService`

Worker runner 形态建议：

```go
type Runner struct {
    consumer  queue.MessageJobConsumer
    processor WorkerProcessor
    logger    *slog.Logger
}

func (r *Runner) Start(ctx context.Context) error
```

- [ ] **Step 4: 重新运行 bootstrap/app 测试并确认通过**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/bootstrap ./internal/app ./internal/worker -run 'Test.*Async|TestWorkerRunner'
```

Expected:
- PASS

- [ ] **Step 5: 提交**

```bash
git add internal/bootstrap internal/app internal/worker/runner.go internal/worker/runner_test.go
git add cmd
git commit -m "update: Wire async message execution mode"
```

### Task 6: 续约、重试、指标与文档收尾

**Files:**
- Modify: `internal/worker/session_lock.go`
- Modify: `internal/worker/agent_message_job_processor.go`
- Modify: `internal/observability/...`
- Modify: `docs/frontend-api.md`
- Create: `docs/eval/async-message-worker.md`

- [ ] **Step 1: 写失败测试，覆盖锁续约、可重试失败、指标记录**

覆盖：
- 长任务处理中锁能续约
- 临时错误能转成 `retryable_failed`
- 超过最大重试后转成 `failed`
- 记录 job latency 和失败计数

- [ ] **Step 2: 运行 worker/observability 测试并确认失败**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/worker ./internal/observability -run 'Test.*Retry|Test.*Renew|Test.*Metrics'
```

Expected:
- 因续约、重试、metrics 未实现而失败

- [ ] **Step 3: 实现锁续约、重试和基础指标**

必须新增或确认：
- `message_jobs_queued_total`
- `message_jobs_processing_total`
- `message_jobs_completed_total`
- `message_jobs_failed_total`
- `message_job_latency_seconds`
- `session_lock_acquire_failures_total`

- [ ] **Step 4: 更新接口文档和使用文档**

文档至少更新：
- `POST /sessions/{id}/messages` 现在返回 `202`
- 新增 `GET /messages/{id}`
- 前端轮询模式说明

- [ ] **Step 5: 重新运行相关测试并确认通过**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/worker ./internal/observability ./internal/transport/http/handler
```

Expected:
- PASS

- [ ] **Step 6: 全量验证**

Run:

```bash
GOCACHE=/tmp/go-build go test ./...
git diff --check
```

Expected:
- 全量测试通过
- 无格式问题

- [ ] **Step 7: 提交**

```bash
git add internal/worker internal/observability docs/frontend-api.md docs/eval/async-message-worker.md
git commit -m "update: Finalize async message worker flow"
```

## Spec 覆盖检查

本计划已覆盖 spec 中的核心要求：

- 异步 `POST /messages`
- `message_jobs` 状态机
- RabbitMQ 任务投递和消费
- Redis session 串行锁
- Worker 池执行真实 Agent
- 结果查询接口
- 监控、重试、续约、回滚开关

未纳入本计划的内容与 spec 保持一致，明确延后：

- SSE / WebSocket
- fast/heavy lane
- 模型分层降时延

## 交付完成定义

本计划全部完成后，系统应满足：

- `POST /sessions/{id}/messages` 正常情况下 `P95 < 300ms`
- 同 session 并发消息不会并发执行
- 不同 session 可由多 worker 并行处理
- 前端可通过 `GET /messages/{id}` 获取处理状态和最终回复
- worker 崩溃后任务不会永久卡死
- 失败和重试不会重复写入 assistant message
