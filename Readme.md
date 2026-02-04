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
- [ ] **OneBot V11 接入 (QQ Bot)** - *Next Step*
- [ ] 更多 DND 5E 规则集成的 Prompt 优化

## 🤝 贡献
欢迎提交 Issue 或 PR 来改进这个项目！特别是 Prompt Engineering 方面的建议。
