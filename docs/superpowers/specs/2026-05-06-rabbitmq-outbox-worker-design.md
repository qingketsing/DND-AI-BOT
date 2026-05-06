# RabbitMQ + Outbox 独立 Worker 异步消息处理设计

## 目标

将当前“同步消息处理链路”升级为“网关接单、数据库持久化、Outbox 发布、RabbitMQ 分发、独立 Worker 消费”的正式异步架构，目标包括：

- 支持 `100+` 并发消息请求进入系统
- 保证同一 `session` 严格串行
- 保证用户消息一旦持久化，就一定存在可追踪的处理命运
- 支持 RabbitMQ 重复投递、Worker 崩溃恢复和可重试失败
- 将 HTTP 接单能力与后台推理能力彻底解耦

本设计解决的是**入口并发承载、一致性、恢复能力和可扩展性**，不直接解决单条消息本身耗时较长的问题。

## 当前问题

当前异步消息骨架已经具备：

- `POST /sessions/{id}/messages -> 202`
- `GET /messages/{id}`
- `message_jobs`
- Redis `session lock`
- 后台 `MessageJobProcessor`

但仍有以下关键缺陷：

1. **入队一致性不完整**
   - 当前顺序是“写 user message -> 写 job -> publish”
   - 任一步失败，都可能留下“消息已显示，但永远不会被处理”的悬空状态

2. **当前队列仍是进程内 channel**
   - 不是 RabbitMQ 真正的跨进程消息队列
   - 无法支撑独立扩容和真实恢复语义

3. **锁没有续约闭环**
   - 长任务超过 TTL 时，同一 `session` 可能被第二个 worker 并发处理

4. **可重试失败没有真正闭环**
   - `retryable_failed` 只是状态，不构成可靠重试链路

5. **reply 关联方式脆弱**
   - 当前依赖“用户消息后面的第一条 agent message”
   - 遇到重试、系统消息、重复回复时不可靠

如果继续在现有骨架上直接接 RabbitMQ 而不补足上述点，系统会从“可跑”变成“更容易出错”。

## 范围

本设计包含：

- `message + job + outbox_event` 事务写入
- RabbitMQ 真正的发布与消费
- 独立 `worker` 进程
- Redis 会话级租约锁与续约
- `message_jobs` 状态机定稿
- `outbox_events` 表和 dispatcher
- retry / dead-letter / stale job recovery 设计
- `reply_to_message_id` 显式关联

本设计不包含：

- SSE / WebSocket 推送
- front-end UI 改造
- fast/heavy lane 多队列分层
- 模型层优化和时延优化
- 跨集群、多地域部署

## 运行形态

### 组件拆分

系统拆成两个正式进程：

1. **`dnd-app`**
   - 提供 HTTP API
   - 鉴权
   - 会话读写
   - 在事务中写入 user message / message_job / outbox_event
   - 提供 `GET /messages/{id}` 状态查询
   - 内置 Outbox Dispatcher

2. **`dnd-worker`**
   - 独立进程
   - 消费 RabbitMQ
   - 抢 Redis `session lock`
   - 执行 `MessageJobProcessor`
   - 调 Agent / RAG / tools
   - 写回 assistant message 和 job 状态

### 为什么不把 consumer 融进 app

如果 RabbitMQ consumer 长期融进 app，会带来：

- HTTP 网关和重推理负载耦合
- app 和 worker 无法独立扩容
- 故障面扩大
- 以后仍需经历一次拆分

因此本设计正式采用：

- app 负责接单和查状态
- worker 负责消费和执行

## 核心原则

1. `POST /messages` 返回 `202` 的语义是：**消息、任务和待发布事件已可靠持久化**
2. 同一 `session` 任意时刻只能有一个 worker 真正执行业务逻辑
3. 不同 `session` 可由多个 worker 并行处理
4. PostgreSQL 是业务真相
5. RabbitMQ 是分发与削峰工具，不是最终状态真相
6. Redis 是租约锁，不是业务真相
7. 所有异步执行必须幂等

## 数据模型

### messages

保留现有 `messages` 结构，并补充显式回复关联：

- `id`
- `session_id`
- `user_id`
- `role`
- `content`
- `status`
- `reply_to_message_id`（仅 assistant message 使用）
- `created_at`
- `updated_at`

规则：

- user message 在事务内写入
- assistant message 由 worker 成功处理后写入
- assistant message 必须带 `reply_to_message_id`

### message_jobs

保留并扩展现有 `message_jobs` 表：

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

### outbox_events

新增 `outbox_events` 表，用于可靠发布 RabbitMQ 事件。

建议字段：

- `id`
- `aggregate_type`
- `aggregate_id`
- `event_type`
- `payload_json`
- `status`
- `attempt_count`
- `last_error`
- `created_at`
- `published_at`
- `updated_at`

推荐含义：

- `aggregate_type`: 固定 `message_job`
- `aggregate_id`: `job_id`
- `event_type`: 首版固定 `message_job_queued`
- `status`: `pending / published / failed`

建议索引：

- `idx_outbox_events_status_created_at`
- `idx_outbox_events_aggregate_id`

## 状态机

### message_jobs 状态

- `queued`
- `published`
- `processing`
- `completed`
- `retryable_failed`
- `failed`
- `cancelled`

### 允许状态迁移

- `queued -> published`
- `published -> processing`
- `processing -> completed`
- `processing -> retryable_failed`
- `processing -> failed`
- `retryable_failed -> published`
- `retryable_failed -> failed`
- `queued -> cancelled`

### 不允许的迁移

- `completed -> *`
- `failed -> processing`
- `cancelled -> processing`

## Outbox 模式

### 事务边界

`POST /sessions/{id}/messages` 在一个数据库事务中同时执行：

1. 写 user message
2. 写 `message_job(status=queued)`
3. 写 `outbox_event(status=pending, event_type=message_job_queued)`

三者要么一起成功，要么一起失败。

### 202 的正式语义

返回 `202 Accepted` 表示：

> 该消息已经被系统可靠保存，并且拥有可恢复的后续处理链路。

不要求 RabbitMQ 在返回前已经收到消息。

### Outbox Dispatcher

由 `dnd-app` 内部启动 dispatcher，周期性扫描：

- `status = pending`
- 或 `status = failed and attempt_count < max`

Dispatcher 流程：

1. 读取待发布 outbox event
2. publish 到 RabbitMQ
3. 成功后标记 `published`
4. 同时推动 `message_job` 从 `queued -> published`
5. 失败时增加 `attempt_count` 和 `last_error`

## RabbitMQ 设计

### Exchange / Queue

- exchange: `agent.message`
- queue: `agent.message.default`
- routing key: `message.process`

### Payload

仅传递最小必要字段：

```json
{
  "job_id": "job_xxx",
  "message_id": "msg_xxx",
  "session_id": "session_xxx",
  "user_id": "user_xxx",
  "attempt": 1,
  "queued_at": "2026-05-06T10:00:00Z"
}
```

约束：

- 不把完整上下文、`game_state`、`session_memory` 放进 MQ
- worker 一律以数据库当前状态为准

## Session Lock 设计

### 锁键

```text
session:{session_id}:processing_lock
```

### 锁值

JSON 序列化的所有者信息：

```json
{
  "job_id": "job_xxx",
  "worker_id": "worker-3"
}
```

### 加锁、续约、释放

- 获取：`SET NX EX ttl`
- 续约：`CompareAndExpire`
- 释放：`CompareAndDelete`

### 参数建议

- TTL：`180s`
- heartbeat：`30s`

### 失锁语义

一旦 worker 续约失败，说明锁所有权可能已经丢失。此时：

- 当前 worker 不再提交最终业务状态
- 将 job 标记为 `retryable_failed`
- 交给 recovery 机制重新投递

## 幂等性设计

### 幂等键

`job_id` 是异步执行的主幂等键。

### 规则

1. worker 开始处理前先读取 `message_job`
2. 如果 job 已经是 `completed`，直接 ack MQ，不再重复执行
3. assistant message 写入时必须带 `reply_to_message_id`
4. 若同一 job 发生重复投递，不允许重复写两条 assistant reply

### reply 关联

正式采用：

- `assistant_message.reply_to_message_id = user_message.id`

不再依赖“用户消息后面的第一条 agent message”这种隐式查找。

## Worker 执行流程

1. 从 RabbitMQ 收到 payload
2. 根据 `job_id` 读取 `message_job`
3. 若 job 已 `completed`，直接 ack
4. 抢 `session lock`
5. 成功后标记 job 为 `processing`
6. 启动 heartbeat 续约锁
7. 读取 session / game_state / encounter / session_memory
8. 调用 Agent / RAG / tools
9. 写入 assistant message（带 `reply_to_message_id`）
10. 将 job 标记为 `completed`
11. 释放锁
12. ack MQ

如果出错：

- 可重试错误：标 `retryable_failed`，进入重试链路
- 不可重试错误：标 `failed`

## retry / dead-letter / recovery

### 可重试错误

- LLM timeout
- 临时 5xx
- Redis 临时错误
- RabbitMQ 临时错误
- session save 临时失败

### 不可重试错误

- session 不存在
- message 不存在
- 用户越权
- 明显数据损坏

### 重试策略

- 最大重试次数：`3`
- 退避：`30s / 2m / 10m`

### 超限策略

- 超过最大重试次数后转 `failed`
- 首版可以先用应用层重投
- 第二版再接 RabbitMQ dead-letter queue

### Stale Processing Recovery

由 `dnd-worker` 或单独后台任务扫描：

- `status = processing`
- `started_at` 早于阈值，如 `10m`
- 对应锁不存在或已不属于该 job

命中后：

- 标记为 `retryable_failed`
- 走重新发布流程

## API 设计

### 发送消息

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

### 查询状态

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
    "attempt_count": 1
  },
  "assistant_reply": null
}
```

完成后：

```json
{
  "message_id": "msg_xxx",
  "session_id": "session_xxx",
  "status": "completed",
  "job": {
    "id": "job_xxx",
    "status": "completed",
    "latency_ms": 38211
  },
  "assistant_reply": {
    "message_id": "msg_reply_xxx",
    "reply_to_message_id": "msg_xxx",
    "content": "门把手在你手中缓缓转动……"
  }
}
```

## 观测指标

至少需要：

- `message_jobs_queued_total`
- `message_jobs_processing_total`
- `message_jobs_completed_total`
- `message_jobs_retryable_failed_total`
- `message_jobs_failed_total`
- `message_job_latency_seconds`
- `outbox_events_pending_total`
- `outbox_publish_failures_total`
- `session_lock_acquire_failures_total`
- `session_lock_renew_failures_total`
- `worker_retry_total`

## 风险与取舍

### 风险 1：RabbitMQ 一时不可用

通过 Outbox 模式规避“消息已写入但任务丢失”。

### 风险 2：长任务锁过期

通过 heartbeat 续约规避。

### 风险 3：重复投递

通过 `job_id` 幂等和 `reply_to_message_id` 显式关联规避。

### 风险 4：processing 卡死

通过 stale job recovery 补偿。

## 实施顺序

1. 新增 `outbox_events` 表和模型
2. 将异步入队改为事务内写 `message + job + outbox`
3. 实现 Outbox Dispatcher
4. 实现 RabbitMQ publisher / consumer
5. 新增独立 `cmd/dnd-worker`
6. 将 `MessageJobProcessor` 切换到 RabbitMQ 消费链路
7. 增加 `reply_to_message_id`
8. 增加锁续约
9. 增加 retry / recovery
10. 补观测指标

## 验收标准

1. `POST /messages` 在异步模式下 `P95 < 300ms`
2. RabbitMQ 短暂不可用时，不出现悬空用户消息
3. 同一 `session` 的两条消息不会并发执行
4. Worker 重启或崩溃后，未完成 job 能恢复
5. 重复消费不会写出重复 assistant reply
6. `GET /messages/{id}` 总能给出可靠状态

## 定稿结论

正式采用以下方案：

- 使用 `Outbox` 解决入队一致性
- 使用独立 `dnd-worker` 进程消费 RabbitMQ
- `dnd-app` 只负责接单、事务持久化、状态查询和 Outbox Dispatcher
- 使用 Redis 租约锁 + heartbeat 续约
- 使用 `job_id` + `reply_to_message_id` 保证幂等和回复关联
- 使用 `retryable_failed + recovery` 而不是单纯依赖一次消费成功
