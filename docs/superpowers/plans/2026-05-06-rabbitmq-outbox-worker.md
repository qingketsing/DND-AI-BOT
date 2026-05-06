# RabbitMQ + Outbox 独立 Worker 异步消息处理实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前“进程内异步骨架”升级为正式的 `Outbox + RabbitMQ + 独立 Worker` 架构，消除入队一致性缺口、补齐锁续约和重试恢复，并为后续高并发消息处理打下稳定基础。

**Architecture:** `dnd-app` 负责 HTTP 接单、事务内写 `user message + message_job + outbox_event`、消息状态查询与 outbox dispatcher；`dnd-worker` 作为独立进程消费 RabbitMQ，使用 Redis `session lock` 与 heartbeat 续约保证同一会话串行，执行 Agent 并写回 assistant reply；PostgreSQL 是业务真相，RabbitMQ 负责分发，Redis 负责租约互斥。

**Tech Stack:** Go, PostgreSQL, RabbitMQ, Redis, 现有 Agent Runtime, 现有 HTTP Handler / DTO / Service / Repository 模式

---

### Task 1: Outbox 数据模型与 Repository

**Files:**
- Create: `internal/model/outbox_event.go`
- Create: `internal/repository/outbox_event.go`
- Create: `internal/repository/postgres/outbox_event_store.go`
- Create: `internal/repository/postgres/outbox_event_store_impl.go`
- Create: `internal/repository/postgres/outbox_event_store_test.go`
- Modify: `migrations/013_create_outbox_events.sql`
- Modify: `internal/repository/errors.go`

- [ ] **Step 1: 写 outbox migration 和 repository 的失败测试**

覆盖：
- `outbox_events` 表结构
- `pending / published / failed` 状态读取
- `GetPending`、`MarkPublished`、`MarkFailedAttempt`

建议表结构：

```sql
CREATE TABLE outbox_events (
    id TEXT PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload_json JSONB NOT NULL,
    status TEXT NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_outbox_events_status_created_at
    ON outbox_events(status, created_at);
CREATE INDEX idx_outbox_events_aggregate_id
    ON outbox_events(aggregate_id);
```

- [ ] **Step 2: 运行失败测试并确认 Red**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/repository/postgres -run 'TestOutboxEvent|TestMigrations'
```

- [ ] **Step 3: 实现 Outbox 模型与 Repository**

最小接口：

```go
type OutboxEventRepository interface {
    Create(ctx context.Context, event model.OutboxEvent) error
    GetPending(ctx context.Context, limit int) ([]model.OutboxEvent, error)
    MarkPublished(ctx context.Context, id string, publishedAt time.Time) error
    MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, lastError string) error
}
```

- [ ] **Step 4: 重新运行测试并确认 Green**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/repository/postgres -run 'TestOutboxEvent|TestMigrations'
```

- [ ] **Step 5: 提交**

```bash
git add internal/model/outbox_event.go internal/repository/outbox_event.go
git add internal/repository/postgres/outbox_event_store.go internal/repository/postgres/outbox_event_store_impl.go
git add internal/repository/postgres/outbox_event_store_test.go migrations/013_create_outbox_events.sql
git commit -m "update: Add outbox persistence model"
```

### Task 2: 事务化异步入队与 reply 显式关联

**Files:**
- Modify: `internal/model/session.go`
- Modify: `internal/service/async_message_service.go`
- Modify: `internal/service/async_message_service_test.go`
- Modify: `internal/repository/message_job.go`
- Modify: `internal/repository/session.go`
- Modify: `internal/transport/http/dto/async_message.go`
- Modify: `internal/transport/http/handler/session_handler_test.go`

- [ ] **Step 1: 写失败测试，覆盖事务内写入与 reply 关联字段**

覆盖：
- 一次入队同时持久化：
  - user message
  - `message_job`
  - `outbox_event`
- publish 从 service 中移除
- `assistant_reply` 支持 `reply_to_message_id`

- [ ] **Step 2: 运行失败测试并确认 Red**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/service ./internal/transport/http/handler -run 'TestAsyncMessageService|TestSessionHandler'
```

- [ ] **Step 3: 实现事务化入队和显式 reply 关联**

要求：
- `AsyncMessageService.EnqueueMessage()` 不再直接 publish
- 在单一事务内写：
  - user message
  - `message_job(status=queued)`
  - `outbox_event(status=pending)`
- assistant message 增加 `reply_to_message_id`

- [ ] **Step 4: 重新运行测试并确认 Green**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/service ./internal/transport/http/handler -run 'TestAsyncMessageService|TestSessionHandler'
```

- [ ] **Step 5: 提交**

```bash
git add internal/model/session.go internal/service/async_message_service.go internal/service/async_message_service_test.go
git add internal/repository/message_job.go internal/repository/session.go
git add internal/transport/http/dto/async_message.go internal/transport/http/handler/session_handler_test.go
git commit -m "update: Add transactional async message enqueue"
```

### Task 3: RabbitMQ Publisher / Consumer 与 Outbox Dispatcher

**Files:**
- Modify: `internal/queue/message_job.go`
- Create: `internal/queue/rabbitmq_publisher.go`
- Create: `internal/queue/rabbitmq_consumer.go`
- Create: `internal/queue/rabbitmq_publisher_test.go`
- Create: `internal/queue/rabbitmq_consumer_test.go`
- Create: `internal/service/outbox_dispatcher.go`
- Create: `internal/service/outbox_dispatcher_test.go`

- [ ] **Step 1: 写失败测试，覆盖 RabbitMQ 发布和 outbox dispatcher**

覆盖：
- exchange / queue / routing key 正确
- payload JSON 编解码
- dispatcher 从 `pending` 读取并 publish
- 成功后标 `published`
- 失败后增加 `attempt_count`

- [ ] **Step 2: 运行失败测试并确认 Red**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/queue ./internal/service -run 'TestRabbitMQ|TestOutboxDispatcher'
```

- [ ] **Step 3: 实现 RabbitMQ 发布/消费与 Outbox Dispatcher**

要求：
- publisher 使用：
  - exchange `agent.message`
  - queue `agent.message.default`
  - routing key `message.process`
- dispatcher 周期扫描 outbox 并发布
- 成功发布后：
  - outbox 标 `published`
  - job 由 `queued -> published`

- [ ] **Step 4: 重新运行测试并确认 Green**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/queue ./internal/service -run 'TestRabbitMQ|TestOutboxDispatcher'
```

- [ ] **Step 5: 提交**

```bash
git add internal/queue internal/service/outbox_dispatcher.go internal/service/outbox_dispatcher_test.go
git commit -m "update: Add RabbitMQ and outbox dispatcher"
```

### Task 4: Worker 锁续约、幂等与恢复状态机

**Files:**
- Modify: `internal/worker/session_lock.go`
- Modify: `internal/worker/session_lock_test.go`
- Modify: `internal/worker/message_job_processor.go`
- Modify: `internal/worker/message_job_processor_test.go`
- Modify: `internal/repository/postgres/message_job_store_impl.go`

- [ ] **Step 1: 写失败测试，覆盖锁续约、幂等和状态迁移**

覆盖：
- 处理长任务时会启动 heartbeat 续约
- 续约失败时停止提交最终状态
- 已完成 job 重复消费时直接退出
- `reply_to_message_id` 写回正确
- `published -> processing -> completed`
- `processing -> retryable_failed`

- [ ] **Step 2: 运行失败测试并确认 Red**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/worker ./internal/repository/postgres -run 'TestMessageJobProcessor|TestSessionLock|TestMessageJobStore'
```

- [ ] **Step 3: 实现锁续约、幂等检查和更严格状态机**

要求：
- TTL 建议 `180s`
- renew 间隔建议 `30s`
- 续约失败后不再提交最终状态
- processor 开始前检查 job 是否已 `completed`
- 使用 `reply_to_message_id` 而不是“找下一条 agent message”

- [ ] **Step 4: 重新运行测试并确认 Green**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/worker ./internal/repository/postgres -run 'TestMessageJobProcessor|TestSessionLock|TestMessageJobStore'
```

- [ ] **Step 5: 提交**

```bash
git add internal/worker internal/repository/postgres/message_job_store_impl.go
git commit -m "update: Harden async worker locking and idempotency"
```

### Task 5: 独立 Worker 进程与 App 集成改造

**Files:**
- Create: `cmd/dnd-worker/main.go`
- Create: `internal/bootstrap/rabbitmq.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/async_message_runtime.go`
- Modify: `internal/bootstrap/runtime.go`
- Modify: `compose.yaml`
- Modify: `compose.prod.yaml`

- [ ] **Step 1: 写失败测试，覆盖 app/worker 进程职责拆分**

覆盖：
- app 不再使用 in-process queue 执行消息
- app 启动 outbox dispatcher
- worker 启动 RabbitMQ consumer
- 异步模式下缺 RabbitMQ 依赖时明确失败

- [ ] **Step 2: 运行失败测试并确认 Red**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/app ./internal/bootstrap -run 'TestNewApp|TestAsync'
```

- [ ] **Step 3: 实现独立 worker 进程**

要求：
- 新增 `cmd/dnd-worker`
- `dnd-app` 负责：
  - HTTP
  - 状态查询
  - outbox dispatcher
- `dnd-worker` 负责：
  - RabbitMQ consumer
  - `MessageJobProcessor`
- 结构上移除对 in-process queue 的长期依赖

- [ ] **Step 4: 重新运行测试并确认 Green**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/app ./internal/bootstrap ./cmd/dnd-worker
```

- [ ] **Step 5: 提交**

```bash
git add cmd/dnd-worker internal/bootstrap internal/app compose.yaml compose.prod.yaml
git commit -m "update: Add standalone async message worker"
```

### Task 6: retry / stale job recovery / observability

**Files:**
- Create: `internal/service/message_job_recovery.go`
- Create: `internal/service/message_job_recovery_test.go`
- Modify: `internal/observability/...`
- Modify: `internal/service/outbox_dispatcher.go`
- Modify: `internal/worker/message_job_processor.go`

- [ ] **Step 1: 写失败测试，覆盖 retry 与 stale recovery**

覆盖：
- `retryable_failed` 能重新进入 `published`
- stale `processing` job 被回收
- outbox publish 失败会累计 attempt
- 关键 metrics 会打点

- [ ] **Step 2: 运行失败测试并确认 Red**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/service ./internal/worker -run 'TestMessageJobRecovery|TestOutboxDispatcher|TestMessageJobProcessor'
```

- [ ] **Step 3: 实现恢复闭环与指标**

要求：
- `retryable_failed` 重试最多 `3` 次
- stale `processing` job 恢复阈值如 `10m`
- 增加：
  - `outbox_events_pending_total`
  - `message_jobs_retryable_failed_total`
  - `session_lock_renew_failures_total`
  - `message_job_latency_seconds`

- [ ] **Step 4: 重新运行测试并确认 Green**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/service ./internal/worker
```

- [ ] **Step 5: 提交**

```bash
git add internal/service internal/worker internal/observability
git commit -m "update: Add async job recovery and observability"
```

### Task 7: 验证与收尾

**Files:**
- Modify: `docs/eval/...`（如有）
- Modify: `README.md`（如有必要补充架构说明）

- [ ] **Step 1: 运行核心测试**

Run:

```bash
GOCACHE=/tmp/go-build go test ./internal/model ./internal/repository/postgres ./internal/queue ./internal/service ./internal/worker ./internal/app ./internal/bootstrap ./cmd/dnd-worker
```

- [ ] **Step 2: 运行 Docker 级手工验证**

验证：
- `POST /sessions/{id}/messages -> 202`
- `GET /messages/{id}` 可查询状态
- worker 消费 RabbitMQ 后能写回 reply
- RabbitMQ 短暂不可用时，不出现悬空 user message

- [ ] **Step 3: 更新文档**

补充：
- 架构图
- app / worker / RabbitMQ / Redis / Postgres 分工
- 运行方式和环境变量

- [ ] **Step 4: 提交**

```bash
git add README.md docs
git commit -m "update: Document RabbitMQ outbox worker architecture"
```
