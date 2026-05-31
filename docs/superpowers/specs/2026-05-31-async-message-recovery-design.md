# 异步消息恢复闭环实现设计

## 背景

当前异步消息链路已经具备以下能力：

- `message_jobs` 任务状态机
- `outbox_events` 可靠待发布事件
- 事务化 `user message + message_job + outbox_event` 入队
- Outbox Dispatcher 发布 RabbitMQ
- `MessageJobProcessor` 执行 job
- Redis `session lock` 与 heartbeat 续约
- assistant reply 的 `reply_to_message_id / source_job_id` 幂等约束

当前最关键的缺口是：**异步任务进入异常中间态后，还没有完整自动恢复闭环**。

最典型的问题是：

```text
job 已经进入 processing
worker 中途崩溃 / 进程退出 / 模型调用卡死
job 永久停在 processing
前端一直查询不到 completed / failed
```

此外，当前还需要补齐：

- `retryable_failed` 到期后重新入队
- `outbox=published` 但 `job=queued` 的状态补齐
- outbox 发布失败后的固定延迟重试
- dispatcher/recovery 可观测性

## 目标

本设计目标是为异步消息链路补齐最小可用恢复能力：

- 防止 `processing` job 永久卡死
- 让 `retryable_failed` job 能按固定 `30s` 策略重新进入处理链路
- 让 outbox 发布失败不会立即空转重试
- 修复 `outbox_events.status=published` 但 `message_jobs.status=queued` 的状态撕裂
- 保证 recovery 动作幂等、保守、可观测

## 非目标

本设计不包含：

- step 级 durable execution
- workflow engine
- RabbitMQ DLQ / delayed exchange
- SSE / WebSocket 推送
- 多队列优先级调度
- 跨集群分布式恢复协调
- 对 Agent Runtime 本身的模型延迟优化

## 核心原则

1. PostgreSQL 是状态真相源。
2. RabbitMQ 是分发通道，不是业务状态真相。
3. Redis lock 只用于判断 session 级执行租约是否仍然被当前 job 持有。
4. Recovery 只能修复异步任务状态，不直接推进游戏业务状态。
5. Recovery 必须保守，宁可晚恢复，也不要误恢复仍在正常运行的 worker。
6. Recovery 操作必须幂等，重复扫描不能造成重复 reply 或重复状态推进。

## 需要恢复的状态

### 1. Outbox 发布失败

状态：

```text
outbox_events.status = failed
```

含义：

- 待发布事件曾经发布 RabbitMQ 失败
- 任务还没有可靠进入消息通道

恢复动作：

- 等 `next_retry_at <= now`
- 由 Outbox Dispatcher 重新 publish
- 成功后：
  - `outbox_events.status = published`
  - `message_jobs.status = published`
- 失败后：
  - `attempt_count + 1`
  - `last_error = error`
  - `next_retry_at = now + 30s`

### 2. Outbox 已发布但 job 仍 queued

状态：

```text
outbox_events.status = published
message_jobs.status = queued
```

含义：

- RabbitMQ 大概率已经收到消息
- outbox 已经标记发布成功
- 但 job 状态没有补成 `published`

恢复动作：

- 将 `message_jobs.status` 从 `queued` 修复为 `published`

这个动作不重新发 RabbitMQ，只修复 PostgreSQL 内部状态。

### 3. processing job 卡死

状态：

```text
message_jobs.status = processing
started_at / updated_at 超过 stale threshold
```

恢复前必须检查 Redis lock：

- 如果 lock 仍存在，且 owner 的 `job_id` 等于当前 job：不恢复
- 如果 lock 不存在：恢复
- 如果 lock 存在但 owner 不是当前 job：恢复
- 如果 Redis 不可用：跳过本轮恢复，fail closed

恢复动作：

```text
processing -> retryable_failed
last_error_code = stale_processing
last_error_message = "processing job exceeded stale threshold and session lock is not owned by this job"
next_retry_at = now + 30s
```

### 4. retryable_failed job 到期

状态：

```text
message_jobs.status = retryable_failed
next_retry_at <= now
attempt_count < max_attempts
```

恢复动作：

- 在同一个 PostgreSQL 事务中：
  - `message_jobs.status = queued`
  - 清空 `finished_at`
  - 创建新的 `outbox_event(status=pending)`

随后由现有 Outbox Dispatcher 发布到 RabbitMQ。

如果：

```text
attempt_count >= max_attempts
```

则恢复动作应将 job 转为：

```text
failed
last_error_code = max_attempts_exceeded
```

## 数据模型调整

### message_jobs 新增字段

新增：

```sql
next_retry_at TIMESTAMPTZ NULL
heartbeat_at TIMESTAMPTZ NULL
```

字段含义：

- `next_retry_at`：下一次允许 recovery 重新入队的时间
- `heartbeat_at`：worker 处理期间最近一次处理心跳时间

第一版 stale 判断仍以 Redis lock owner 为主，`heartbeat_at` 用于观测和后续扩展。

索引：

```sql
CREATE INDEX idx_message_jobs_retryable_next_retry_at
    ON message_jobs(status, next_retry_at)
    WHERE status = 'retryable_failed';

CREATE INDEX idx_message_jobs_processing_updated_at
    ON message_jobs(status, updated_at)
    WHERE status = 'processing';
```

### outbox_events 新增字段

新增：

```sql
next_retry_at TIMESTAMPTZ NULL
```

字段含义：

- `failed` outbox event 下一次允许 dispatcher 重发的时间

索引：

```sql
CREATE INDEX idx_outbox_events_dispatch_due
    ON outbox_events(status, next_retry_at, created_at)
    WHERE status IN ('pending', 'failed');
```

## Repository 能力调整

### MessageJobRepository

新增能力：

```go
ListStaleProcessing(ctx context.Context, cutoff time.Time, limit int) ([]model.MessageJob, error)
ListRetryDue(ctx context.Context, now time.Time, limit int) ([]model.MessageJob, error)
MarkRetryScheduled(ctx context.Context, jobID string, now time.Time, nextRetryAt time.Time, code string, message string) error
RequeueRetryableWithOutbox(ctx context.Context, job model.MessageJob, event model.OutboxEvent, now time.Time) error
MarkHeartbeat(ctx context.Context, jobID string, now time.Time) error
RepairPublished(ctx context.Context, jobID string, now time.Time) error
```

其中 `RequeueRetryableWithOutbox` 必须是事务化操作。

### OutboxEventRepository

调整 `GetPending` 语义：

- 读取 `pending`
- 读取 `failed AND next_retry_at <= now`

新增能力：

```go
ListPublishedWithQueuedJobs(ctx context.Context, limit int) ([]OutboxJobRepairCandidate, error)
MarkFailedAttempt(ctx context.Context, id string, failedAt time.Time, nextRetryAt time.Time, lastError string) error
```

`MarkFailedAttempt` 需要写入固定 `30s` 后的 `next_retry_at`。

### SessionLock

新增只读检查能力：

```go
type SessionLockOwner struct {
    Exists   bool
    JobID    string
    WorkerID string
}

Inspect(ctx context.Context, sessionID string) (SessionLockOwner, error)
```

Redis 实现通过 `GET session:{session_id}:processing_lock` 后解析 JSON value。

Recovery 只读这个 owner，不修改锁。

## Recovery 组件设计

新增服务：

```text
internal/service/async_recovery.go
```

核心结构：

```go
type AsyncMessageRecovery struct {
    jobs MessageJobRecoveryRepository
    outbox OutboxRecoveryRepository
    lock worker.SessionLockInspector
    clock func() time.Time
    config AsyncRecoveryConfig
}
```

配置：

```go
type AsyncRecoveryConfig struct {
    Interval time.Duration
    BatchSize int
    RetryDelay time.Duration
    ProcessingStaleAfter time.Duration
}
```

默认值：

```text
interval = 10s
batch_size = 50
retry_delay = 30s
processing_stale_after = 10m
```

### RunOnce 流程

每轮执行顺序：

1. repair published outbox + queued job
2. recover stale processing jobs
3. requeue due retryable_failed jobs

推荐顺序原因：

- 先修复已发布状态，避免误判 queued job
- 再处理卡死 processing
- 最后把到期 retry job 重新写入 outbox

## Stale Processing 具体判定

候选查询：

```sql
SELECT *
FROM message_jobs
WHERE status = 'processing'
  AND updated_at < $cutoff
ORDER BY updated_at ASC
LIMIT $limit;
```

对每个候选：

1. 调 `lock.Inspect(session_id)`
2. 如果 Redis 返回错误：跳过，不恢复
3. 如果 `owner.Exists && owner.JobID == job.ID`：跳过，不恢复
4. 否则标记为 `retryable_failed`

这样可以避免：

- LLM 正在长时间思考
- worker heartbeat 仍然正常
- recovery 误把任务恢复

## Retryable Failed 重新入队

候选查询：

```sql
SELECT *
FROM message_jobs
WHERE status = 'retryable_failed'
  AND next_retry_at IS NOT NULL
  AND next_retry_at <= $now
ORDER BY next_retry_at ASC, updated_at ASC
LIMIT $limit;
```

处理规则：

- `attempt_count >= max_attempts`：标记 `failed`
- `attempt_count < max_attempts`：
  - 事务内将 job 改回 `queued`
  - 创建新的 outbox event

新的 outbox payload 继续使用：

```json
{
  "job_id": "...",
  "message_id": "...",
  "session_id": "...",
  "user_id": "...",
  "attempt": 2,
  "queued_at": "..."
}
```

其中 `attempt` 建议使用 `attempt_count + 1`。

## Outbox Backoff

Outbox 发布失败不应立即每秒重复 publish。

第一版统一使用固定延迟：

```text
next_retry_at = now + 30s
```

Dispatcher 读取条件：

```text
status = pending
OR (status = failed AND next_retry_at <= now)
```

这样可以避免 RabbitMQ 故障时 dispatcher 空转打满日志。

## Published Repair

修复条件：

```text
outbox_events.status = published
outbox_events.aggregate_type = message_job
outbox_events.aggregate_id = message_jobs.id
message_jobs.status = queued
```

修复动作：

```text
message_jobs.status = published
message_jobs.updated_at = now
```

这个 repair 动作可以重复执行，重复执行应无副作用。

## Worker Heartbeat 与 Recovery 的关系

当前 Redis heartbeat 只能证明：

- worker 进程还活着
- 当前 job 仍持有 session lock

它不能证明：

- LLM 一定在正常返回
- 业务逻辑一定还在推进

因此第一版 recovery 判定采用：

```text
processing stale + Redis lock 不存在/不属于当前 job
```

不单独用时间作为恢复依据。

后续如果要更精细，可以让 worker 在每次 heartbeat 成功后同步调用 `jobs.MarkHeartbeat` 更新 `heartbeat_at`。

## API 与前端影响

本设计不新增 HTTP API。

现有：

```text
GET /messages/{message_id}
```

继续根据 `message_jobs.status` 返回：

- `queued`
- `processing`
- `completed`
- `failed`

当 recovery 将 `processing -> retryable_failed -> queued` 后，前端会看到状态回到 `queued` 或保持排队中语义。

## 可观测性

新增日志：

- `async recovery run started`
- `async recovery run finished`
- `published repair applied`
- `stale processing recovered`
- `retryable job requeued`
- `retryable job max attempts exceeded`
- `stale processing skipped because lock owner matches`
- `stale processing skipped because redis unavailable`

建议指标：

- `async_recovery_runs_total{status}`
- `async_recovery_duration_seconds`
- `async_recovery_stale_processing_total`
- `async_recovery_retry_requeued_total`
- `async_recovery_published_repair_total`
- `async_recovery_skipped_total{reason}`

第一版可以先落结构化日志，Prometheus 指标可作为后续增强。

## 配置

新增环境变量：

```env
ASYNC_MESSAGE_RECOVERY_ENABLED=true
ASYNC_MESSAGE_RECOVERY_INTERVAL_MS=10000
ASYNC_MESSAGE_RECOVERY_BATCH_SIZE=50
ASYNC_MESSAGE_RETRY_DELAY_SECONDS=30
ASYNC_MESSAGE_PROCESSING_STALE_SECONDS=600
```

默认：

- app 异步模式开启时，recovery 默认开启
- 非异步模式下不启动 recovery loop

## 运行位置

第一版 recovery loop 可以先跑在 `dnd-app` 内，和 Outbox Dispatcher 一样作为后台 loop。

原因：

- 当前 app 已经拥有 PostgreSQL、Redis、Outbox 依赖
- 独立 worker 进程闭环仍在推进
- 先放 app 内能更快补齐生产风险最高的卡死恢复

后续当独立 `dnd-worker` 完整落地后，可以将 Job Recovery 拆到 worker 或单独 recovery runner。

## 实施任务拆分

### Task 1: 数据模型与 migration

修改：

- `migrations/012_create_message_jobs.sql` 或新增 migration
- `migrations/013_create_outbox_events.sql` 或新增 migration
- `internal/model/message_job.go`
- `internal/model/outbox_event.go`

新增字段：

- `message_jobs.next_retry_at`
- `message_jobs.heartbeat_at`
- `outbox_events.next_retry_at`

### Task 2: Repository 扩展

修改：

- `internal/repository/message_job.go`
- `internal/repository/outbox_event.go`
- `internal/repository/postgres/message_job_store_impl.go`
- `internal/repository/postgres/outbox_event_store_impl.go`
- memory/fake test helpers

覆盖：

- due retry 查询
- stale processing 查询
- retryable requeue + outbox 事务写入
- outbox due dispatch
- published repair

### Task 3: SessionLock Inspect

修改：

- `internal/worker/session_lock.go`
- `internal/worker/session_lock_test.go`

新增：

- `Inspect(ctx, sessionID)`

要求：

- Redis key 不存在返回 `Exists=false`
- value JSON 可解析出 `job_id / worker_id`
- Redis 错误向上返回

### Task 4: Recovery Service

新增：

- `internal/service/async_recovery.go`
- `internal/service/async_recovery_test.go`

覆盖：

- stale processing lock owner 匹配时跳过
- stale processing lock 不存在时恢复
- Redis 错误时跳过
- retryable_failed 到期后重新入队
- attempt 超限后 failed
- published repair 成功

### Task 5: Runtime Loop 与 Bootstrap

新增/修改：

- `internal/app/async_recovery_runtime.go`
- `internal/app/app.go`
- `internal/bootstrap/async_recovery_config.go`
- `cmd/api/main.go`

要求：

- 异步模式下启动 recovery loop
- app close 时停止 loop
- 配置可通过 env 覆盖

### Task 6: Verification

定向测试：

```bash
GOCACHE=/tmp/go-build go test ./internal/worker ./internal/service ./internal/repository/postgres ./internal/app ./internal/bootstrap -run 'Test.*Recovery|Test.*SessionLock|Test.*MessageJob|Test.*Outbox'
```

全量关键测试：

```bash
GOCACHE=/tmp/go-build go test ./internal/worker ./internal/service ./internal/repository/postgres ./internal/app ./internal/bootstrap
```

静态检查：

```bash
git diff --check
```

## 验收标准

功能验收：

- `processing` job 在 lock 不存在时会进入 `retryable_failed`
- lock owner 仍为当前 job 时不会被误恢复
- Redis 不可用时不会恢复 processing job
- `retryable_failed` job 到期后会重新生成 pending outbox event
- 超过 `max_attempts` 的 retryable job 会进入 `failed`
- `outbox=published/job=queued` 会被修复为 `job=published`
- failed outbox event 会在 `30s` 后才重试

安全验收：

- Recovery 不会直接写 assistant reply
- Recovery 不会推进 game state / encounter
- Recovery 不会在 Redis lock owner 匹配时抢跑
- 重复执行 recovery 不会产生重复 reply

可观测验收：

- 每轮 recovery 有结构化日志
- 每个恢复动作有明确 reason
- Redis 不可用时有 skip 日志

## 风险与取舍

### 风险 1: 误恢复仍在运行的 job

控制：

- 不只看时间
- 必须检查 Redis lock owner
- Redis 不可用时跳过

### 风险 2: retryable_failed 重复生成 outbox event

控制：

- `RequeueRetryableWithOutbox` 在事务内检查 job 当前状态仍为 `retryable_failed`
- 只在状态更新成功时创建 outbox event
- 新 outbox event 使用唯一 ID

### 风险 3: Recovery 和 Worker 并发修改同一 job

控制：

- 所有状态更新都带 `WHERE status = ...`
- 依赖 affected rows 判断是否成功
- 状态已变化时视为其它执行者已处理

### 风险 4: Outbox 发布成功但 DB 更新失败

控制：

- published repair 扫描修复 `outbox=published/job=queued`
- 后续可进一步把 `outbox.MarkPublished + job.MarkPublished` 合并到 DB 事务

## 最终结论

本设计优先补齐当前异步链路最高风险的恢复缺口：

```text
stale processing recovery
retryable_failed requeue
outbox failed backoff
published repair
```

它不会把系统升级成完整工作流引擎，但能把当前异步消息链路从“能跑”推进到“卡住后能自动恢复”的阶段。
