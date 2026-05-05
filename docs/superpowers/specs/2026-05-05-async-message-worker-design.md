# 异步 Worker 化消息处理设计

## 目标

将当前同步的 `POST /sessions/{id}/messages` 改造成“快速接单、后台处理、异步回写”的消息处理系统，支持 `100+` 并发消息请求进入系统，同时保证：

- 同一 `session` 严格串行
- 不同 `session` 可并行处理
- 消息处理结果可查询
- 失败可恢复

本设计解决的是**入口并发承载能力**，不直接解决单条消息天然很慢的问题。

## 当前问题

当前消息链路是重同步链路，典型执行路径包括：

1. 读取会话历史
2. 构建系统提示词和上下文
3. 执行 RAG 检索
4. 调用 LLM
5. 调用工具
6. 更新 `game_state` / `encounter` / `session_memory`
7. 可能再次调用 LLM
8. 最终返回 HTTP 响应

结果是：

- 单条消息耗时长，常见在数秒到数十秒
- 请求在 HTTP 链路上长时间阻塞
- 并发请求上来后，入口层会先被拖垮
- 压测时容易测到认证、排队、拒绝路径，而不是真实消息处理能力

如果目标是支撑 `100+` 并发消息请求，继续维持同步 HTTP 模式不可行。

## 范围

本设计包含：

- 消息接口异步化
- `message_jobs` 任务状态机
- RabbitMQ 任务分发
- Redis `session lock`
- Worker 池后台消费
- 结果查询接口
- 基础指标与失败恢复

本设计不包含：

- SSE / WebSocket 首版推送
- fast/heavy lane 分层 worker
- 模型分层与降时延优化
- 跨机房、多地域部署

## 核心原则

1. `POST /messages` 不再等待 Agent 完整执行完成
2. 同一 `session` 任意时刻只能处理一条消息
3. 不同 `session` 可以由多个 worker 并行处理
4. PostgreSQL 是状态真相
5. RabbitMQ 只负责任务分发，不存最终业务状态
6. Redis 只负责短期互斥锁，不存业务真相
7. Worker 扩容能力必须独立于 Gateway

## 总体架构

### 组件

系统拆成四类角色：

1. **Gateway / API 层**
   - 接收前端消息请求
   - 校验身份和会话归属
   - 落库 user message
   - 创建任务并投递队列
   - 快速返回 `queued`

2. **Queue / RabbitMQ**
   - 承接待处理消息任务
   - 解耦 HTTP 接入和 Agent 执行
   - 支持削峰与重试

3. **Worker 池**
   - 消费消息任务
   - 执行 Agent / RAG / tools
   - 回写 assistant message 和任务状态

4. **Result Read Path**
   - 前端轮询消息状态
   - 获取最终 assistant reply

### 并发模型

本系统的并发模型不是“所有任务都并行”，而是：

- **同一 session 串行**
- **不同 session 并行**

这是本设计的关键约束。否则会出现：

- 会话消息乱序
- 战斗状态冲突
- `game_state` 被覆盖
- `session_memory` 漂移

## 执行流程

### 写路径

前端调用：

```http
POST /sessions/{id}/messages
```

Gateway 执行：

1. 校验当前用户
2. 校验 `session` 是否属于当前用户
3. 生成 `message_id`
4. 写入一条 user message
5. 创建一条 `message_job`，状态为 `queued`
6. 发布 RabbitMQ job
7. 返回 `202 Accepted`

### 后台路径

Worker 执行：

1. 从 RabbitMQ 取出一个 job
2. 读取 `message_job`
3. 抢 `session lock`
4. 将 job 标记为 `processing`
5. 读取会话历史、`game_state`、`encounter`、`session_memory`
6. 执行 Agent / RAG / tools
7. 写入 assistant message
8. 将 job 标记为 `completed` 或失败态
9. 释放锁
10. `ack` 当前 MQ 消息

### 读路径

前端通过：

- `GET /messages/{message_id}`
- 或 `GET /sessions/{id}`

查询当前消息处理状态和最终回复。

## 数据模型

### messages

如果现有 `messages` 表已经存在，建议补充或确认这些字段：

- `id`
- `session_id`
- `user_id`
- `role`
- `content`
- `status`
- `created_at`
- `updated_at`

建议语义：

- user message 在入队前创建，默认 `completed`
- assistant message 仅由 worker 处理成功后创建
- 失败时首版不强制创建 assistant 占位消息

### message_jobs

新增 `message_jobs` 表，作为异步消息任务状态机：

- `id`
- `message_id`
- `session_id`
- `user_id`
- `status`
- `attempt_count`
- `max_attempts`
- `worker_id`
- `queued_at`
- `started_at`
- `finished_at`
- `last_error_code`
- `last_error_message`
- `latency_ms`
- `created_at`
- `updated_at`

### 任务状态

- `queued`
- `processing`
- `completed`
- `retryable_failed`
- `failed`
- `cancelled`

### 建议索引

- `idx_message_jobs_status_created_at`
- `idx_message_jobs_session_id_created_at`
- `idx_message_jobs_message_id`
- `idx_message_jobs_user_id_created_at`

## RabbitMQ 设计

### Exchange / Queue

首版使用单队列最小闭环：

- exchange: `agent.message`
- queue: `agent.message.default`
- routing key: `message.process`

### Job Payload

MQ payload 只传最小必要信息：

```json
{
  "job_id": "job_xxx",
  "message_id": "msg_xxx",
  "session_id": "session_xxx",
  "user_id": "user_xxx",
  "attempt": 1,
  "queued_at": "2026-05-05T15:00:00Z"
}
```

原则：

- 不把完整上下文放进 MQ
- 不把 `session_memory` / `game_state` 放进 MQ
- 运行时上下文一律以数据库为准

### 重试策略

首版建议：

- `max_attempts = 3`
- 可重试错误：
  - 外部 LLM timeout
  - 短时 5xx
  - RabbitMQ / Redis / DB 短时异常
- 不可重试错误：
  - session 不存在
  - message 不存在
  - 权限错误
  - 参数错误

## Redis 会话锁设计

### 锁键

```text
session:{session_id}:processing_lock
```

### 锁值

建议存 JSON 或结构化字符串，至少包含：

```json
{
  "job_id": "job_xxx",
  "worker_id": "worker-3",
  "acquired_at": "2026-05-05T15:00:00Z"
}
```

### 加锁语义

使用：

```text
SET key value NX EX 300
```

即：

- 不存在时才允许加锁
- 默认过期时间 300 秒

### 续约

因为单条消息可能执行很久，worker 运行期间必须定期续约，例如每 30 秒续约一次。

### 解锁

释放锁必须 compare-and-delete。禁止直接裸 `DEL`，否则可能误删其他 worker 的新锁。

### 设计目的

Redis 锁只解决一件事：

- 同一 `session` 同时只能有一个 job 进入真正处理阶段

## API 设计

### 发送消息

接口：

```http
POST /sessions/{id}/messages
```

请求：

```json
{
  "content": "继续攻击并结算伤害"
}
```

响应：

```http
202 Accepted
```

```json
{
  "message_id": "msg_xxx",
  "job_id": "job_xxx",
  "session_id": "session_xxx",
  "status": "queued"
}
```

### 查询消息状态

新增：

```http
GET /messages/{message_id}
```

处理中：

```json
{
  "message_id": "msg_xxx",
  "session_id": "session_xxx",
  "status": "processing",
  "job": {
    "id": "job_xxx",
    "status": "processing",
    "attempt_count": 1,
    "queued_at": "2026-05-05T15:00:00Z",
    "started_at": "2026-05-05T15:00:02Z"
  },
  "assistant_reply": null
}
```

完成：

```json
{
  "message_id": "msg_xxx",
  "session_id": "session_xxx",
  "status": "completed",
  "job": {
    "id": "job_xxx",
    "status": "completed",
    "attempt_count": 1,
    "latency_ms": 38211
  },
  "assistant_reply": {
    "message_id": "msg_reply_xxx",
    "content": "门把手在你手中缓缓转动……"
  }
}
```

### 会话详情接口

保留 `GET /sessions/{id}`，但建议在响应中补充：

- 是否存在 `processing` job
- 最近一条 user message 是否尚未完成

## Go 包与接口设计

### 建议新增包

- `internal/agentasync/`
- `internal/queue/`
- `internal/worker/`

### 输入输出模型

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

type AsyncMessageService interface {
	EnqueueMessage(ctx context.Context, input EnqueueMessageInput) (*EnqueueMessageResult, error)
	GetMessageStatus(ctx context.Context, messageID string, userID string) (*MessageStatusResult, error)
}
```

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

### Repository

```go
type MessageJobRepository interface {
	Create(ctx context.Context, job MessageJob) error
	GetByID(ctx context.Context, jobID string) (*MessageJob, error)
	GetByMessageID(ctx context.Context, messageID string) (*MessageJob, error)
	MarkProcessing(ctx context.Context, jobID string, workerID string, startedAt time.Time) error
	MarkCompleted(ctx context.Context, jobID string, finishedAt time.Time, latencyMS int64) error
	MarkRetryableFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error
	MarkFailed(ctx context.Context, jobID string, finishedAt time.Time, errorCode string, errorMessage string) error
	IncrementAttempt(ctx context.Context, jobID string) error
}
```

### Queue

```go
type MessageJobPublisher interface {
	Publish(ctx context.Context, payload MessageJobPayload) error
}

type MessageJobConsumer interface {
	Start(ctx context.Context, handler func(context.Context, MessageJobPayload) error) error
}
```

### Session Lock

```go
type SessionLock interface {
	Acquire(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) (bool, error)
	Renew(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) error
	Release(ctx context.Context, sessionID string, jobID string, workerID string) error
}
```

### Worker Processor

```go
type WorkerProcessor interface {
	ProcessMessageJob(ctx context.Context, payload MessageJobPayload) error
}
```

## Worker 处理流程

建议顺序：

1. 读取 `message_job`
2. 如果 job 已完成，直接 `ack`
3. 抢 `session lock`
4. 成功后将 job 标记为 `processing`
5. 启动锁续约
6. 读取会话历史、`game_state`、`session_memory`
7. 执行 agent runner
8. 写入 assistant message
9. 更新 job 状态为 `completed`
10. 释放锁
11. `ack`

### 抢不到锁

说明同一 `session` 正在处理别的消息。

首版建议：

- 不直接标失败
- 延迟重试或 requeue

### 错误分流

- `ErrSessionBusy`
  - requeue
- 临时外部错误
  - `retryable_failed`
  - 允许重试
- 不可恢复错误
  - `failed`

## 前端交互方案

首版建议采用**轮询**，不直接上 SSE。

前端流程：

1. 用户发送消息
2. 立即渲染 user message
3. 保存后端返回的 `message_id`
4. 每 `1~2s` 请求一次 `GET /messages/{message_id}`
5. `completed` 时展示 assistant reply
6. `failed` 时展示失败占位和重试按钮

优点：

- 实现简单
- 对现有前端侵入较小

## 指标与观测

### 指标

必须新增：

- `message_jobs_queued_total`
- `message_jobs_processing_total`
- `message_jobs_completed_total`
- `message_jobs_failed_total`
- `message_job_latency_seconds`
- `session_lock_acquire_failures_total`
- `session_lock_held_seconds`
- `queue_publish_failures_total`
- `worker_process_failures_total`

### 日志字段

至少记录：

- `job_id`
- `message_id`
- `session_id`
- `worker_id`
- `status`
- `attempt_count`
- `latency_ms`
- `error_code`

## 兼容与迁移

建议迁移顺序：

1. 增加 `message_jobs` 表和 repository
2. 接入 RabbitMQ publisher/consumer
3. 实现 Redis `session lock`
4. 新增异步 `POST /messages`
5. 新增 `GET /messages/{id}`
6. 接入 worker
7. 前端切换到轮询模型
8. 保留旧同步实现一段时间用于回滚

### 回滚策略

建议保留同步实现，通过配置切换：

- `MESSAGE_EXECUTION_MODE=sync|async`

当 worker 或 MQ 出现严重故障时，可以切回同步模式止血。

## 风险与防护

### 主要风险

- 同 session 锁处理不严导致状态乱序
- worker 崩溃后锁残留
- MQ 消费成功但 DB 写回失败
- 前端轮询与会话详情渲染不一致
- 重试导致重复 assistant message

### 防护要求

- assistant message 写入必须幂等
- job 完成态必须可重复检查
- lock release 必须 compare-and-delete
- retry 必须以 `job_id` 作为幂等键

## 实施顺序

### Phase 1

- 数据表和 migration
- repository
- RabbitMQ publisher / consumer
- Redis session lock
- `POST /messages -> 202`
- `GET /messages/{id}`

### Phase 2

- worker 接真实 agent runner
- assistant reply 回写
- 任务状态闭环

### Phase 3

- retry
- heartbeat renew lock
- 监控和报警

### Phase 4

- SSE
- fast/heavy lane
- worker pool 分层

## 验收标准

最小可用闭环标准：

- `POST /messages` 在正常条件下 `P95 < 300ms`
- 同一 `session` 并发两条消息时，后一条不会与前一条并发执行
- 不同 `session` 可由多个 worker 同时处理
- `GET /messages/{id}` 能正确显示 `queued / processing / completed / failed`
- worker crash 后任务不会永久卡死
- assistant 回复不会因重试重复写入多条

## 结论

如果目标是支撑 `100+` 并发消息请求，这套架构是当前最务实、最符合现有技术栈的方案：

- Gateway 轻量接单
- RabbitMQ 解耦和排队
- Redis 保证会话级串行
- Worker 池横向扩展
- PostgreSQL 存储状态真相

它不会让单条消息突然变快，但会把系统从“重同步阻塞链路”转成“可扩展的异步消息处理系统”，为后续真正的吞吐提升和分层优化打下基础。
