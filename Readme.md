# DND AI Dungeon Master Bot

这是一个基于 Go 语言开发的 DND (龙与地下城) 跑团辅助机器人，旨在通过接入大语言模型 (LLM)，如 DeepSeek/OpenAI，来扮演地下城主 (DM) 的角色，为玩家提供沉浸式的文字跑团体验。

本项目目前处于 **Alpha** 阶段，支持 CLI 命令行模式进行测试，并计划支持 OneBot V11 协议以接入 QQ 等即时通讯软件。

## ✨ 核心特性

*   **🧙 AI DM 主持**: 
    *   接入强大的 LLM (推荐 DeepSeek-V3/R1)，根据玩家的描述实时生成剧情。
    *   基于 DND 5E 规则进行裁决，维护游戏世界的逻辑与真实性。
*   **🧠 智能记忆与上下文**:
    *   **滑动窗口机制**: 自动保留最近的对话上下文。
    *   **长期记忆摘要 (Auto-Summarization)**: 每 20 轮对话自动触发剧情总结，确保 AI 即使在长团中也能记住早期的关键剧情。
    *   **动态状态注入**: 每一轮对话都会将当前角色的状态（HP、职业、属性）注入给 AI，防止幻觉。
*   **🎲 真实的骰子与检定**:
    *   内置 `.r` 投骰指令 (标准 DND 骰子表达式，如 `1d20+5`)。
    *   骰子结果作为客观事实传递给 AI，AI 必须严格根据点数描述成功或失败。
*   **⚡ AI 动作协议 (Action Protocol)**:
    *   AI 不仅仅是说话，还能**执行操作**。
    *   可以判定玩家是否受伤并**扣除血量**。
    *   可以为 NPC 进行**暗骰检定**。
    *   **死亡机制**: 当 HP 归零时，AI 会判定角色死亡并将其移除出当前战局。
*   **🛡️ 反作弊与逻辑性**:
    *   严谨的 DM 人格设定，拒绝接受玩家不合逻辑的指令（如“直接给我加满血”）。
    *   所有的状态变更需符合游戏内逻辑。

## 🛠️ 快速开始

### 1. 环境准备
*   Go 1.20+
*   一个兼容 OpenAI 格式的 LLM API Key (推荐 DeepSeek)

### 2. 配置
在项目根目录创建 `.env` 文件：

```env
OPENAI_API_KEY=your_sk_key_here
OPENAI_BASE_URL=https://api.deepseek.com
MODEL_NAME=deepseek-chat
```

### 3. 运行 (CLI 模式)
目前版本主要通过命令行界面进行测试：

```bash
# 安装依赖
go mod tidy

# 编译并运行
go run .
```

### 4. 常用指令
进入 CLI 后，你可以使用以下指令：

- `.st [name] [class] [hp] [str]`: 创建角色卡 (例如 `.st Alice Warrior 20 16`)
- `.show`: 显示当前所有角色状态与背景
- `.bg [description]`: 设置当前场景/背景描述
- `.r [expression]`: 投掷骰子 (例如 `.r 1d20+2`, `.r 4d6`)
- `.reset`: 清空当前会话记忆
- `.exit`: 退出程序
- **[直接输入文本]**: 与 DM AI 进行对话，推进剧情

## 🏗️ 架构设计

*   **Pkg/AI**: 封装 LLM 调用，处理 System Prompt 注入。
*   **Pkg/Session**: 管理会话上下文，实现滑动窗口与自动摘要 (Summarization)。
*   **Pkg/Game**: 管理游戏状态 (Character, HP, Group State)。
*   **Pkg/Dice**: 独立的骰子运算模块。
*   **Main**: CLI 入口，指令解析，消息分发，AI Action 解析器。

## 📝 开发计划
详细进度请参阅 [TODO.md](TODO.md)

- [√] 基础框架与 LLM 接入
- [√] 角色卡与状态管理
- [√] AI Action (自动扣血/投骰)
- [√] 长期记忆 (摘要系统)
- [x] **OneBot V11 接入 (QQ Bot)** - *已支持*
- [ ] 更多 DND 5E 规则集成的 Prompt 优化

## 🚢 部署指南：QQ 机器人 (Docker 一键部署)

我们提供了一套基于 **Docker Compose** 的完整解决方案，包含了 DNDBot 核心服务以及 **NapCat** (一个开源的无头 QQ 客户端，支持各种OneBot协议)。

### 1. 前置准备

*   安装 [Docker](https://www.docker.com/) 和 [Docker Compose](https://docs.docker.com/compose/)。
*   准备一个 **QQ 小号** 用于机器人登录（建议不要使用主号，以防风控）。
*   准备好 DeepSeek 或其他兼容 OpenAI 格式的 **API Key**。

### 2. 配置文件

在项目根目录下创建或修改 `.env` 文件。请务必填入你的真实信息：

```env
# ========================
# AI 模型配置 (必需)
# ========================
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxx
OPENAI_BASE_URL=https://api.deepseek.com
MODEL_NAME=deepseek-chat

# ========================
# QQ 账号配置 (用于 NapCat 自动登录)
# ========================
QQ_ACCOUNT=123456789

# ========================
# 机器人连接配置
# ========================
# 如果使用 Docker Compose 部署，请保持为空或保持默认 (ws://napcat:3001)
# 宿主机网络或本地调试则填写: ws://127.0.0.1:3001
ONEBOT_WS_URL=ws://napcat:3001
# 如果 NapCat 配置了 Token，请在此填写，否则留空
ONEBOT_ACCESS_TOKEN=
```

### 3. 启动服务 (方式 A: Docker)

```bash
docker-compose up -d
```

### 3. 启动服务 (方式 B: Windows 本地运行 - 推荐)
如果 Docker 网络有问题，请直接在 Windows 上运行：

1.  **下载 NapCat**: 前往 [NapCat Release](https://github.com/NapNeko/NapCatQQ/releases) 下载最新 Windows 版本（如 `NapCat.Shell.zip`）。
2.  **运行 NapCat**: 
    * 解压并运行 `NapCat.Shell.exe`。
    * 输入你的 QQ 号，按提示**扫码登录**。
    * **重要配置**: 确保 NapCat 开启了 WebSocket 服务，端口为 **3001**。
      *(NapCat 默认 WebUI 地址为 http://127.0.0.1:6099/webui，进入网络配置页面，添加一个 WebSocket Server，端口设为 3001，Host设为 0.0.0.0，并在 DNDBot 的 .env 中确保 ONEBOT_WS_URL=ws://127.0.0.1:3001)*
3.  **启动机器人**:
    * 直接双击 `run_local.bat` (如果你想进 CLI 模式)。
    * 或者在终端运行 `go run main.go`。

### 4. 扫码登录 QQ (Docker 模式)

服务启动后，NapCat 容器需要你扫码登录 QQ。

1.  查看日志获取二维码：
    ```bash
    docker logs -f napcat
    ```
2.  终端屏幕上会显示一个二维码，**请使用手机 QQ 扫描该二维码** 并确认登录。
3.  看到 "登录成功" 或类似日志后，机器人即准备就绪。

### 5. 群聊使用方法

将你的机器人 QQ 拉入群聊，然后即可开始交互：

*   **@机器人 + 文字**: 与 DM 进行对话，推进剧情。
    *   *示例*: `@DNDBot 我向酒馆老板打听最近关于“失落矿坑”的传闻。`
*   **`.r [表达式]`**: 投掷骰子。
    *   *示例*: `.r 1d20+3`
    *   机器人会回复你的点数，并将其记录到 DM 的后台日志中，供下一次剧情裁决使用。
*   **DM 主动判定**:
    *   如果 DM 认为你需要进行检定（如感知、敏捷豁免），它会在回复中说明，并可能直接帮你投暗骰，或者要求你自己投。
    *   如果有战斗发生，Bot 会自动计算并在后台扣除你的 HP。

---

## 🛠️ 本地开发与 CLI 模式

如果你不需要连接 QQ，只想在终端里快速测试 AI 逻辑：

1.  确保 `.env` 中 **不要** 设置 `ONEBOT_WS_URL` (或将其留空)。
2.  运行程序：
    ```bash
    go run .
    ```
3.  你将进入命令行交互模式，可以直接输入对话。

### CLI 独有指令
*   `.st [name] [class] [hp] [str]`: 创建一张临时角色卡。
*   `.show`: 查看当前状态。
*   `.reset`: 重置记忆。

## ⚠️ 常见问题

**Q: Docker 拉取镜像一直失败？**
A: 这是通过由于国内网络环境导致 Docker Hub 访问受阻。请尝试配置国内镜像源（如阿里云、网易云），或者在本地开启代理。

**Q: 扫码后 NapCat 提示异地登录或风控？**
A: 新账号或长期未使用的账号容易触发风控。建议先在手机上挂几天，或者在同一网络环境下扫码。

**Q: 机器人没有回复群消息？**
A: 
1. 检查 Logs: `docker logs dndbot` 查看是否有报错。
2. 确认你是否 **@了机器人** (当前逻辑要求群聊必须 @ 才会触发回复)。
3. 检查 API Key 余额是否充足。
