# Next.js 前端设计交接稿

## 目标

为当前 Go 后端 Agent 系统设计一个独立的 Next.js 前端项目，整体风格参考 Discord 聊天页面，主色使用紫色。  
这份文档用于后续在另一个前端项目中继续落地，不直接在本仓库实现前端代码。

当前后端能力已经具备：

- `POST /sessions`
- `GET /sessions/{id}`
- `POST /sessions/{id}/messages`

后端职责：

- Session 管理
- Agent Runtime
- DeepSeek 模型调用
- `search_rules`
- `search_lore`

前端职责：

- 创建并持有当前 session
- 展示聊天消息
- 发送消息
- 呈现 loading / error
- 提供接近 Discord 的聊天体验

---

## 总体定位

这个前端不是“AI 工具台”，而是一个 **DM 聊天客户端**。

用户应该感受到：

- 像在 Discord 某个频道里和 DM 对话
- 页面偏沉浸、偏游戏感
- AI 感弱，聊天室感强
- 布局清晰、信息密度高

不建议做成：

- 通用问答机器人页面
- 白底紫按钮的普通 AI SaaS 风格
- 过于夸张的奇幻 UI

目标是：

**Discord 的结构感 + 跑团世界的气氛 + 紫色主题**

---

## 推荐技术方案

建议新项目使用：

- `Next.js` App Router
- `TypeScript`
- `React`
- `CSS Modules` 或 `Tailwind CSS`

如果没有现成设计系统，建议优先：

- `Next.js + TypeScript + Tailwind`

原因：

- 页面结构清晰
- 聊天布局实现快
- 后续状态管理简单

---

## 页面结构

建议只做一个主页面：

- `/`

页面布局模仿 Discord 的三栏思路，但第一版收敛成两栏半：

### 1. 左侧窄栏：服务器感导航

作用：

- 提供视觉上的 Discord 氛围
- 放品牌、世界名、快捷入口

第一版内容建议：

- 顶部 Logo / 世界名
- 当前 campaign 图标
- 一个高亮中的“DM 频道”

这一栏先不做真实导航逻辑，主要是视觉骨架。

宽度建议：

- `72px`

### 2. 中间栏：频道/信息栏

作用：

- 展示当前聊天上下文
- 强化“频道”感

第一版内容建议：

- 频道标题：`# dm-table`
- 副标题：一行说明，例如：
  - `Ask rules, lore, and actions in-character.`
- 当前 `session id`
- 连接状态
  - `Connected`
  - `Thinking...`
  - `Error`

宽度建议：

- `240px ~ 280px`

### 3. 右侧主栏：聊天主区域

这是核心。

包含：

- 顶部频道头部
- 消息列表
- 输入区

头部建议包含：

- `# dm-table`
- 一句频道说明
- 右上角一个小状态点

---

## 视觉风格

## 配色方向

主视觉参考 Discord，但不要照抄默认色值。

建议色板：

- 背景主色：深灰蓝
- 次级背景：更深的紫灰
- 主主题色：紫色
- 强调色：亮紫 / 偏霓虹紫
- 文本主色：偏冷白
- 次级文本：灰紫色

建议变量：

```css
:root {
  --bg-app: #1e1f29;
  --bg-sidebar: #171821;
  --bg-panel: #202332;
  --bg-panel-2: #2a2e42;
  --bg-input: #262a3b;
  --text-primary: #f3f4fb;
  --text-secondary: #aeb3c8;
  --text-muted: #7f86a3;
  --accent: #7c5cff;
  --accent-hover: #9277ff;
  --accent-soft: rgba(124, 92, 255, 0.16);
  --border: rgba(255, 255, 255, 0.08);
  --danger: #ff6b81;
  --success: #4dd4ac;
}
```

### 风格重点

- 整体暗色
- 紫色只做重点，不要把整个页面刷成紫色
- 通过层次背景制造 Discord 的面板感
- 聊天气泡不要做成移动端 IM 风格，应更像频道消息流

---

## 消息流设计

不要做左右气泡式微信聊天。  
要更接近 Discord 的“频道消息列表”。

### 推荐消息结构

每条消息显示：

- 头像
- 名称
- 时间
- 内容

### 用户消息

- 名称：用户昵称
- 头像：固定圆形字母头像
- 内容区域不需要明显气泡，可直接是消息块

### DM 消息

- 名称：`DM Agent`
- 头像：紫色徽记或带图案的圆形头像
- 内容区域背景略深一点
- 可在左侧用细紫线或浅紫底强调

### 系统状态消息

例如：

- `Thinking...`
- `Session created`
- `Connection lost`

应当更轻、更细、更弱化，不要和主消息竞争视觉焦点。

---

## 输入区设计

输入区固定在底部，参考 Discord 文本输入框。

建议包含：

- 大号输入框
- 发送按钮
- 可选提示文字

样式建议：

- 输入容器圆角较大
- 背景使用 `--bg-input`
- 发送按钮使用紫色强调
- 输入框内部不要边框感太重

占位文案建议：

- `Send a message to the DM...`

### 行为

- 页面首次加载时自动创建 session
- 创建成功后才能输入
- 发送中时：
  - 禁用发送按钮
  - 显示 loading 状态
- 回车发送，`Shift+Enter` 换行

---

## 交互流程

### 页面初始化

1. 前端加载
2. 调 `POST /sessions`
3. 保存 `session.id`
4. 状态切为可聊天

### 发送消息

1. 用户输入内容
2. 调 `POST /sessions/{id}/messages`
3. 以后端返回的 `history` 为准重绘消息流
4. 自动滚动到底部

### 错误处理

至少区分：

- session 创建失败
- 消息发送失败
- 后端 500
- 网络错误

展示方式：

- 输入区上方一条红色细条错误提示
- 不要用浏览器默认 alert

---

## API 对接建议

前端项目中建议封装：

### 环境变量

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

### 类型建议

```ts
export type SessionResponse = {
  id: string
  channel: string
  history: HistoryRecord[]
  created_at: string
  updated_at: string
}

export type HistoryRecord = {
  id: string
  user: {
    id: string
    name: string
  }
  message: {
    content: string
  }
  sequence: number
  source: 'user' | 'agent' | 'system'
  created_at: string
}
```

### API 方法建议

```ts
export async function createSession(): Promise<SessionResponse>

export async function getSession(sessionId: string): Promise<SessionResponse>

export async function sendMessage(
  sessionId: string,
  input: {
    userId: string
    userName: string
    content: string
  }
): Promise<SessionResponse>
```

---

## 推荐组件拆分

建议最少拆这些：

### `app/page.tsx`

职责：

- 页面级状态
- 初始化 session
- 调 API
- 持有 `history`
- 传给子组件渲染

### `components/app-shell.tsx`

职责：

- 整体三栏布局

### `components/server-rail.tsx`

职责：

- 左侧窄栏

### `components/channel-sidebar.tsx`

职责：

- 中间栏信息面板

### `components/chat-header.tsx`

职责：

- 右侧顶部频道头部

### `components/message-list.tsx`

职责：

- 渲染消息流
- 区分 `user` / `agent` / `system`

### `components/message-input.tsx`

职责：

- 输入
- 回车发送
- loading 状态

---

## 状态设计

页面状态建议只保留最小集合：

```ts
type ChatPageState = {
  sessionId: string | null
  history: HistoryRecord[]
  loading: boolean
  creatingSession: boolean
  error: string | null
}
```

不建议第一版就引入复杂全局状态库。  
页面级 `useState` / `useReducer` 足够。

---

## 文案风格

整体页面文案要统一成：

- 像频道
- 像 DM 桌面
- 不像 AI SaaS

推荐：

- 页面标题：`DM Table`
- 频道名：`# dm-table`
- 侧栏名称：`Campaign`
- 状态：
  - `Connected`
  - `Thinking`
  - `Waiting on the DM`

不推荐：

- `AI Assistant`
- `Ask anything`
- `Your smart chatbot`

---

## 第一版完成标准

做到以下几点即可：

- 打开页面自动创建 session
- 页面是 Discord 风格布局
- 紫色主题完成
- 能发送消息
- 能展示 Go 后端的真实回复
- loading / error 状态可见

---

## 第二版再考虑的内容

这些不要在第一版就做：

- 会话列表
- 用户身份系统
- 战斗面板
- 角色卡面板
- Markdown 富文本渲染优化
- 消息编辑 / 重发
- 工具调用过程可视化

---

## 给后续 Codex 的直接任务说明

可以把下面这段直接复制给另一个前端项目里的 Codex：

```md
请基于 Next.js App Router + TypeScript 实现一个聊天页面，风格参考 Discord。

要求：
- 使用深色主题
- 主强调色使用紫色
- 页面布局参考 Discord：左侧窄栏 + 中间频道信息栏 + 右侧聊天主栏
- 页面加载时自动调用 Go 后端 `POST /sessions`
- 发送消息时调用 `POST /sessions/{id}/messages`
- 以后端返回的 history 为准渲染消息流
- 消息流采用 Discord 风格，不要做左右气泡 IM 风格
- DM 消息应有明显但克制的紫色强调
- 提供 loading 和 error 状态
- 第一版不要做登录、多会话、复杂控制台

后端 API base URL 从 `NEXT_PUBLIC_API_BASE_URL` 读取。
```

---

## 一句话总结

这版前端应该做成：

**一个 Discord 风格、紫色主题、面向 DM 对话的 Next.js 聊天客户端，直接消费现有 Go Agent API。**
