# DND-AI-BOT

AI-native D&D DM 后端服务，提供长会话记忆、结构化游戏状态、混合检索 RAG 和 Web API。

前端仓库：<https://github.com/qingketsing/DND_fe>

## About

- **定位**：面向文字跑团场景的 Agent 后端，而不是完整游戏客户端
- **核心能力**：会话与认证、DM Agent Runtime、规则/设定检索、角色与战斗状态持久化、长会话记忆与摘要
- **当前阶段**：可运行的正式产品雏形，重点在稳定性、可观测性和长会话连续性

## Architecture

请求进入 Web API 后，后端会完成：

1. 用户认证、会话读取与限流
2. 自动预加载最近消息、`game_state`、`encounter_state`、`session_memory`
3. 按意图选择 fast / primary 模型
4. 通过 Agent Runtime 调用工具、检索规则与设定
5. 将角色、场景、战斗和摘要写回 PostgreSQL / Redis

## Core Features

- **Agent Runtime**：支持多步工具调用、模型分层和降级回复
- **Structured State**：角色、草稿、场景、战斗、长期记忆均有结构化状态
- **Hybrid RAG**：支持 lexical / hybrid 检索，hybrid 模式基于 pgvector
- **Long Session Memory**：支持滑动窗口、摘要压缩、自动预热上下文
- **Observability**：支持结构化日志、模型/工具耗时与基础 metrics
- **Deployment Ready**：支持 Docker Compose、Caddy 反向代理、PostgreSQL 自动备份

## Tech Stack

- Go
- PostgreSQL
- Redis
- pgvector
- Docker / Docker Compose
- Agent Runtime / Tool Calling
- Hybrid Retrieval / RAG

## Repository Layout

```text
DND-AI-BOT/
├── cmd/                    # API、评测等入口
├── configs/                # 配置模板与评测配置
├── data/                   # 规则/设定数据与处理中间产物
├── deploy/                 # 生产代理与部署脚本
├── docs/                   # 设计、计划、运行文档
├── internal/
│   ├── agent/              # client / runtime / tools / prompt / routing
│   ├── app/                # 应用装配与依赖注入
│   ├── game/               # rules / combat / state 等领域模型
│   ├── repository/         # PostgreSQL / Redis / composite repository
│   ├── service/            # 会话、状态、记忆、认证等服务
│   └── transport/          # HTTP handler / middleware / router
├── migrations/             # 数据库迁移
├── ops/                    # 备份与运维脚本
└── scripts/                # 检索、部署、测试脚本
```

## Quick Start

### 1. Prepare env

至少需要配置：

```env
MODEL_PROVIDER=
MODEL_NAME=
MODEL_API_KEY=
MODEL_BASE_URL=

POSTGRES_DSN=
POSTGRES_ADDR=
REDIS_ADDR=
HTTP_ADDR=:8080
```

如果启用 hybrid 检索，还需要：

```env
SEARCH_BACKEND=hybrid
EMBEDDING_PROVIDER=
EMBEDDING_MODEL=
EMBEDDING_API_KEY=
EMBEDDING_BASE_URL=
EMBEDDING_DIM=1024
```

### 2. Run locally

```bash
docker compose up -d --build
```

默认 API 地址：

```text
http://localhost:8080
```

### 3. Run tests

```bash
GOCACHE=/tmp/go-build go test ./...
```

## Retrieval Modes

```env
SEARCH_BACKEND=lexical
```

可选值：

- `lexical`
- `hybrid`

## Production Notes

- 生产环境建议通过 `compose.prod.yaml` + Caddy 反向代理暴露服务
- Cookie、CORS、Trusted Proxies、Metrics 访问边界必须明确配置
- PostgreSQL 自动备份由 `postgres-backup` 服务负责

关键变量示例：

```env
APP_DOMAIN=api.your-domain.com
CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com
AUTH_COOKIE_SECURE=true
TRUSTED_PROXIES=172.16.0.0/12
METRICS_ALLOWED_CIDRS=172.16.0.0/12
```

## Repository Hygiene

以下内容不应提交到仓库：

- `.cache/go-build/`
- 本地二进制，例如 `main`
- `reports/` 下的评测产物
- `.env` 等本地敏感配置

## GitHub Topics

建议仓库 About / topics 使用：

`golang`, `agent`, `rag`, `pgvector`, `postgresql`, `redis`, `docker`, `dnd`, `game-backend`, `ai-backend`

如果仓库启用了 Settings App，可直接使用 [`.github/settings.yml`](/home/qingke/DND-AI-BOT/.github/settings.yml) 同步这些元数据。
