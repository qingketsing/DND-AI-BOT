# DND Bot Alpha 版本开发计划

## Phase 1: 基础设施搭建 (Infrastructure)
- [√] **环境配置**
    - [√] 初始化 Go Module (`go mod init dndbot`)
    - [√] 安装依赖管理工具 (如 `godotenv` 用于环境变量)
    - [√] 选择并安装 OneBot V11 Go SDK (如 `ZeroBot` 或自行封装 WebSocket)
    - [√] 创建基础 `.env` 配置文件

## Phase 2: 核心 AI 模块 (AI Core)
- [√] **LLM 接入**
    - [√] 选择并申请 LLM API (DeepSeek/OpenAI/Zhipu)
    - [√] 安装 Go OpenAI SDK (推荐 `github.com/sashabaranov/go-openai`)
    - [√] 编写 `pkg/ai/client.go`: 封装 API 调用函数
    - [√] 编写简单的 `_test.go` 验证 API 连通性
- [√] **DM 人格设定**
    - [√] 编写 System Prompt (系统提示词)
    - [√] 定义 DM 的说话风格、限制条件 (字数、语气)

## Phase 3: 记忆与会话管理 (Memory & Context) - *核心难点*
- [√] **会话存储**
    - [√] 设计 `SessionManager` 结构体 (Struct)
    - [√] 实现“滑动窗口”机制 (Slice 操作，只保留最近 N 轮对话)
    - [√] 使用 `Map` 区分不同群组 (Group ID) 的上下文，注意并发安全 (`sync.RWMutex`)
- [√] **Context 组装**
    - [√] 实现 Prompt 拼接逻辑: `System Prompt` + `User Info` + `History`

## Phase 4: 简易数值系统 (Stats - Alpha Simplicity)
- [√] **极简角色卡**
    - [√] 定义数据结构 (Struct): `Name`, `Class`, `HP`, `Str` (力量)
    - [√] 实现 `.st` (或者 `.new`) 指令创建角色
    - [√] 实现 `.show` 指令查看角色状态
- [√] **动态状态注入**
    - [√] 在每次请求 LLM 时，自动读取当前群内玩家的 HP/状态
    - [√] 将状态文本插入到 System Prompt 之后，让 AI "看见" 玩家状态

## Phase 5: 骰子与行动判定 (Dice & Logic)
- [√] **基础投骰指令**
    - [√] 实现 `.r [表达式]` (例如 `.r 1d20`)
    - [√] 使用 Go `math/rand` (或 `crypto/rand`) 库计算结果，而非让 AI 生成
- [√] **AI 判定流**
    - [√] 用户输入行动 -> 代码投骰 -> 结果 + 用户行动 -> 发送给 AI
    - [√] 提示词工程: 教 AI 如何根据骰点结果描述成功或失败

## Phase 6: 高级功能与优化 (Advanced Features & Optimization)
- [√] **AI Action Protocol (指令协议)**
    - [√] 设计 `<dnd_action>` JSON 格式
    - [√] 实现解析器: 支持 `roll` (代投) 和 `hp` (改血量)
    - [√] 整合到 DM System Prompt
- [√] **游戏逻辑增强**
    - [√] 死亡判定: HP <= 0 时自动移除角色并公告
    - [√] 反作弊机制: Prompt 强化，拒绝玩家直接修改数据的请求
    - [√] 系统提示优化: 将客观结果 (Roll) 标记为 User 侧的 `【系统提示】`，强制 AI 读取
- [√] **长期记忆系统 (Long-term Memory)**
    - [√] Session 结构升级: 增加 Summary 字段
    - [√] 自动总结: 每 20 条消息触发 Summarization
    - [√] 上下文注入: 将 Summary 动态拼接到 System Prompt

## Phase 7: QQ Bot 集成与提示词升级 (Integration & Prompt 2.0)
- [√] **OneBot V11 接入 (实装)**
    - [√] 手动实现 WebSocket Client (`pkg/bot/onebot.go`)
    - [√] 改造 `main.go`: 从 CLI 循环切换为事件驱动模型
    - [√] 实现群消息监听: 过滤指令 (`.`) 与普通对话 (`@Bot`)
    - [√] 消息处理逻辑复用 (Command Handler & Chat Logic)
    - [√] **[Hotfix]** 修复 OneBot V11 Array 消息格式解析错误 (`interface{}` vs `string`)
    - [√] **[Hotfix]** 增强 Docker 日志输出 (显示收发的原始消息)
    - [√] 系统自检指令 (`.check`) 实装
- [√] **多角色分离实现**
    - [√] .show 等指令的角色id参数传入的实现
- [√] **角色信息参数，背景故事快照保存** 
    - [√] .snapshot 指令
- [ ] **提示词工程 (Prompt Engineering) 2.0**
    - [ ] **动态背景系统**: 允许用户上传/修改更长的世界设定文档 (World Bible)
    - [ ] **风格化 DM**: 支持切换不同的主持风格 (e.g. `.style horror` 恐怖风格, `.style comedy` 搞笑风格)
    - [ ] **规则增强**: 在 System Prompt 中增加对“优势/劣势”、“借机攻击”等进阶规则的理解

## Phase 8: 稳定性与持久化 (Stability & Persistence)
- [ ] **数据持久化**
    - [ ] 接入 SQLite/PostgreSQL 存储角色卡与会话历史 (当前为内存存储，重启即失)
    - [ ] 保存 Campaign 状态
- [ ] **图片生成 (多模态)**
    - [ ] 接入 SD/Midjourney 接口生成场景图
    - [ ] 发送图片消息 (`[CQ:image]`)
- [ ] **运维监控**
    - [ ] 增加心跳检测机制
    - [ ] 自动重连优化 (Exponential Backoff)



