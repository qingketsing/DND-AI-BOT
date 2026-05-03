# DND Game Bot

## 什么是DND Game Bot?

一个基于Golang开发的龙与地下城跑团程序服务端。本服务端提供 Web 服务接口，实现游戏运行，提供基础的游戏逻辑检定和服务支持。

通过Agent代理游戏DM(即游戏主持人)，降低玩家游玩难度，侧重游戏逻辑检定和互动，同时提供了后端api，可以通过外部调用或者修改自己的api key，使用提示词攻击防御能力更高的AI或文本处理能力更强的AI。

## 如何工作？

可以通过网页端直接进行游戏。内部逻辑是通过接入AI API，收集用户消息，对消息进行准确分析，判定，并给予回复。检定逻辑则是通过将消息进行处理，再通过pgvector对原著设定进行查询和判定。不再使用上一版测试的全文检定，而是通过更为准确的段落查询检定，减少tokens消耗，显著提高反应速度。

请求进入 Web API 后，后端会完成：

1. 用户认证、会话读取与限流
2. 自动预加载最近消息、`game_state`、`encounter_state`、`session_memory`
3. 按意图选择 fast / primary 模型
4. 通过 Agent Runtime 调用工具、检索规则与设定
5. 将角色、场景、战斗和摘要写回 PostgreSQL / Redis

## 游戏规则，回复和当前工作重心

目前本游戏服务端仅支持文字游戏，并不支持实际游戏场景渲染。如果对游戏画面有更高的要求，那么这个项目可能并不适合您。

或许在遥远的未来我们可以加入游戏场景渲染功能，不过这并不是我们目前的工作方向。

项目当前聚焦于：

- Web 后端 API
- 会话与认证管理
- Agent Runtime 与工具调用
- 规则 / 设定检索
- 游戏状态持久化
- 长会话记忆与摘要

## 文档和社交媒体

- 本项目前后端分离，本项目只为后端部分。前端仓库详见 <https://github.com/qingketsing/DND_fe>
- 相关文档参考 `docs` 文件夹
- 如果有相关建议和游戏逻辑 bug，请在 Issue 中提出

## 关键特征

当前版本为 V0 版本，正在持续更新中。

- Web端即可游玩：支持直接加入网站进行游戏，注册后即可游戏
- 语句向量化检索：通过对原设定中相应部分的准确判定，降低tokens消耗，显著提高了回复准确率
- 持久化存储：在退出或长时间不游戏后自动退出，但并不会清空进度，具备持久化存储的功能
- Agent Runtime：支持多步工具调用、模型分层和降级回复
- Structured State：角色、草稿、场景、战斗、长期记忆均有结构化状态
- Hybrid RAG：支持 `lexical` / `hybrid` 检索，`hybrid` 模式基于 pgvector
- Long Session Memory：支持滑动窗口、摘要压缩、自动预热上下文
- Observability：支持结构化日志、模型/工具耗时与基础 metrics

## 技术栈

- Golang
- PostgreSQL
- Redis
- pgvector
- Docker / Docker Compose
- Agent Runtime
- Retrieval / RAG

## 文件树介绍

```text
DND-AI-BOT/
├── README.md
├── cmd/
│   ├── api/                # API 服务入口
│   └── soak_eval/          # 长会话评测入口
├── configs/
├── docs/
├── migrations/
├── internal/
│   ├── app/                # 应用装配、依赖注入、启动初始化
│   ├── config/             # 配置读取
│   ├── transport/
│   │   └── http/
│   │       ├── handler/
│   │       ├── router/
│   │       └── dto/
│   ├── agent/              # Agent 编排层，不只是 AI client
│   │   ├── client/
│   │   ├── intent/
│   │   ├── prompt/
│   │   ├── runtime/
│   │   ├── router/
│   │   └── tools/
│   ├── game/               # 领域核心
│   │   ├── combat/
│   │   ├── rules/
│   │   ├── session/
│   │   └── state/
│   ├── retrieval/          # 检索系统
│   │   ├── chunking/
│   │   ├── embedding/
│   │   └── search/
│   ├── repository/
│   │   ├── postgres/
│   │   ├── redis/
│   │   └── vector/
│   └── service/            # 业务服务
├── ops/
├── scripts/
└── data/
```

## 环境变量

最少需要配置以下变量。

### 聊天模型

```env
MODEL_PROVIDER=
MODEL_NAME=
MODEL_API_KEY=
MODEL_BASE_URL=
MODEL_TIMEOUT_SECONDS=60

# 可选：快速模型。未配置时逐项回退到 MODEL_*。
FAST_MODEL_PROVIDER=
FAST_MODEL_NAME=
FAST_MODEL_API_KEY=
FAST_MODEL_BASE_URL=
FAST_MODEL_TIMEOUT_SECONDS=

# 可选：会话摘要模型。未配置时逐项回退到 MODEL_*。
SUMMARY_MODEL_PROVIDER=
SUMMARY_MODEL_NAME=
SUMMARY_MODEL_API_KEY=
SUMMARY_MODEL_BASE_URL=
SUMMARY_MODEL_TIMEOUT_SECONDS=
```

### 检索后端

```env
SEARCH_BACKEND=lexical
```

可选值：

- `lexical`
- `hybrid`

### Embedding

当 `SEARCH_BACKEND=hybrid` 或执行 `build_hybrid_index` 时，需要：

```env
EMBEDDING_PROVIDER=
EMBEDDING_MODEL=
EMBEDDING_API_KEY=
EMBEDDING_BASE_URL=
EMBEDDING_DIM=1024
EMBEDDING_BATCH_SIZE=16
EMBEDDING_TIMEOUT_SECONDS=120
```

### 基础依赖

```env
POSTGRES_DSN=
POSTGRES_ADDR=
REDIS_ADDR=
HTTP_ADDR=:8080
```

### 生产安全配置

生产环境必须配置明确的前端域名、HTTPS Cookie 和 metrics 访问边界。

```env
CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com
CORS_ALLOWED_METHODS=GET,POST,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,X-Request-ID
CORS_ALLOW_CREDENTIALS=true
HTTP_MAX_BODY_BYTES=1048576
TRUSTED_PROXIES=127.0.0.1/32,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
AUTH_COOKIE_SECURE=true
AUTH_COOKIE_DOMAIN=
AUTH_COOKIE_SAMESITE=lax
METRICS_ENABLED=true
METRICS_ALLOWED_CIDRS=127.0.0.1/32,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
METRICS_BEARER_TOKEN=
```

如果 `/metrics` 必须暴露给公网入口，必须设置 `METRICS_BEARER_TOKEN`，否则建议只允许内网 CIDR 访问。

### 工具调用观测

模型调用和工具调用耗时默认只记录超过 1 秒的成功调用，失败调用始终记录。调试时可以打开全量日志。

```env
AGENT_LATENCY_BREAKDOWN_LOG_MODE=slow
AGENT_LATENCY_BREAKDOWN_THRESHOLD_MS=10000
RUNTIME_MODEL_CALL_LOG_MODE=slow
RUNTIME_MODEL_CALL_LOG_THRESHOLD_MS=1000
TOOL_CALL_LOG_MODE=slow
TOOL_CALL_LOG_THRESHOLD_MS=1000
```

可选模式：

- `off`：不记录成功调用日志，失败仍记录。
- `slow`：记录超过阈值的成功调用，失败始终记录。
- `all`：记录所有成功和失败调用。

### 接口限流

限流使用 Redis 作为分布式计数后端。默认开启；如果本地单测或开发环境没有 Redis，应用装配会跳过限流服务。

```env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_LOGIN_IP_LIMIT=10
RATE_LIMIT_LOGIN_IP_WINDOW=1m
RATE_LIMIT_LOGIN_ACCOUNT_LIMIT=5
RATE_LIMIT_LOGIN_ACCOUNT_WINDOW=5m
RATE_LIMIT_REGISTER_IP_LIMIT=5
RATE_LIMIT_REGISTER_IP_WINDOW=1h
RATE_LIMIT_REGISTER_EMAIL_LIMIT=3
RATE_LIMIT_REGISTER_EMAIL_WINDOW=1h
RATE_LIMIT_MESSAGE_USER_LIMIT=30
RATE_LIMIT_MESSAGE_USER_WINDOW=1m
RATE_LIMIT_MESSAGE_SESSION_LIMIT=20
RATE_LIMIT_MESSAGE_SESSION_WINDOW=1m
RATE_LIMIT_MESSAGE_IP_LIMIT=60
RATE_LIMIT_MESSAGE_IP_WINDOW=1m
```

### PostgreSQL 自动备份

项目提供 `postgres-backup` Docker 服务，每天 UTC 03:00 执行一次 `pg_dump -Fc`，备份文件默认写入宿主机：

```text
./backups/postgres
```

手动备份：

```bash
docker compose run --rm postgres-backup /ops/postgres-backup/backup.sh
```

恢复指定备份前建议先停 API：

```bash
docker compose stop app
docker compose run --rm postgres-backup /ops/postgres-backup/restore.sh /backups/<backup-file>.dump
docker compose start app
```

### 生产反向代理 / HTTPS

本地开发仍然使用默认 compose：

```bash
docker compose up -d --build
```

本地访问：

```text
http://localhost:8080
```

生产部署使用 Caddy 覆盖文件：

```bash
docker compose -f compose.yaml -f compose.prod.yaml up -d --build
```

生产模式只发布 Caddy 的 `80/443`，`app:8080`、`postgres:5432`、`redis:6379` 只在 Docker 内网暴露。

生产必须配置：

```env
APP_DOMAIN=api.your-domain.com
CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com
AUTH_COOKIE_SECURE=true
TRUSTED_PROXIES=172.16.0.0/12
METRICS_ALLOWED_CIDRS=172.16.0.0/12
```

检查生产 compose 合并结果：

```bash
sh scripts/deploy/test_prod_proxy_config.sh
```

## 仓库清洁约定

以下内容不应提交到仓库：

- `.cache/go-build/`
- 本地二进制，例如 `main`
- `reports/` 下的评测产物
- `.env` 等本地敏感配置

## 社区贡献者

- ❤️ 感谢每位社区贡献者~

<a href="https://github.com/qingketsing/DND-AI-BOT/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=qingketsing/DND-AI-BOT" />
</a>

## GitHub Topics

建议仓库 About / topics 使用：

`golang`, `agent`, `rag`, `pgvector`, `postgresql`, `redis`, `docker`, `dnd`, `game-backend`, `ai-backend`

如果仓库启用了 Settings App，可直接使用 [`.github/settings.yml`](/home/qingke/DND-AI-BOT/.github/settings.yml) 同步这些元数据。

## LOVE & PEACE
