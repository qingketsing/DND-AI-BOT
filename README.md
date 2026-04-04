# DND Game Bot

## What is DND Game Bot

一个基于Golang开发的龙与地下城跑团程序服务端。本服务端提供web服务和one bot接口，实现游戏运行，提供基础的游戏逻辑检定和服务支持。

通过Agent代理游戏DM(即游戏主持人)，降低玩家游玩难度，侧重游戏逻辑检定和互动，同时提供了后端api，可以通过外部调用或者修改自己的api key，使用提示词攻击防御能力更高的AI或文本处理能力更强的AI。

## How it Works?

可以通过网页端或者将机器人拉入Q群的方式，直接进行游戏。内部逻辑是通过接入AI API，收集用户消息，对消息进行准确分析，判定，并给予回复。检定逻辑则是通过将消息进行处理，再通过pgvector对原著设定进行查询和判定。不再使用上一版测试的全文检定，而是通过更为准确的段落查询检定，减少tokens消耗，显著提高反应速度。

## Game rule , Reply and Words only.

目前本游戏服务端仅支持文字游戏，并不支持实际游戏场景渲染。如果对游戏画面有更高的要求，那么这个项目可能并不适合您。 :sob:

或许在遥远的未来我们可以加入游戏场景渲染功能，不过这并不是我们目前的工作方向。

## Docs and Social media group

- 本项目前后端分离，本项目只为后端部分。目前前端正在计划开工中(马上就有了，别催了QAQ)
- 相关文档参考docs文件夹
- 如果有相关建议和游戏逻辑bug请在Issue中提出，如果想要加入开发团队，可以加入Q群：318633428

## Key Features

当前版本为V0版本，正在持续更新中...

- Web端即可游玩：支持直接加入网站进行游戏，注册后即可游戏
- 语句向量化检索：通过对原设定中相应部分的准确判定，降低tokens消耗，显著提高了回复准确率
- 持久化存储：在退出或长时间不游戏后自动退出，但并不会清空进度，具备持久化存储的功能
- QQ bot引入：已经在排期了！马上就会有！

## Techno Stack

- Golang
- PostgreSQL
- Redis
- pgvector
- Next.js

## File Tree Introduction

```text
DND-AI-BOT/
├── README.md
├── cmd/
│   └── api/
│       └── main.go
├── configs/
├── docs/
├── migrations/
├── deployments/
├── internal/
│   ├── app/                # 应用装配、依赖注入、启动初始化
│   ├── config/             # 配置读取
│   ├── transport/
│   │   └── http/
│   │       ├── handler/
│   │       ├── router/
│   │       └── dto/
│   ├── agent/              # Agent 编排层，不只是 AI client
│   │   ├── orchestrator/
│   │   ├── tools/
│   │   ├── prompt/
│   │   └── client/
│   ├── game/               # 领域核心
│   │   ├── engine/
│   │   ├── rules/
│   │   ├── state/
│   │   └── session/
│   ├── retrieval/          # 检索系统
│   │   ├── chunking/
│   │   ├── embedding/
│   │   ├── search/
│   │   └── rerank/
│   ├── repository/
│   │   ├── postgres/
│   │   ├── redis/
│   │   └── vector/
│   └── model/              # 核心实体/聚合根/公共领域模型
├── pkg/
└── scripts/
```

## Community Co-Building

- ❤️ Contributors

<a href="https://github.com/qingketsing/DND-AI-BOT/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=qingketsing/DND-AI-BOT" />
</a>
