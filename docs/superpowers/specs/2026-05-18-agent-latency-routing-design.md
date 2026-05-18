# Agent 延迟优化：模型分流与 ReAct 轮数控制设计

## 目标

当前 Agent 回复慢的主要原因不是 HTTP、数据库或 Redis，而是多轮 ReAct 中反复等待 LLM 返回。一次 pro 模型调用如果需要 `8s~15s`，当一次请求进入 `4~6` 轮 ReAct 时，整体回复很容易接近一分钟。

本设计目标是将当前“所有请求默认走多轮 ReAct”的链路，改造成：

- 简单请求走 fast path，不进入 ReAct
- 简单请求使用 `flash` 模型单次生成
- 复杂请求使用 `pro` 模型和受控 ReAct
- ReAct 最大轮数限制在 `2~3` 步
- 通过高层上下文构建减少工具调用轮数
- 通过日志和评测验证延迟、成功率和 step 数变化

本设计解决的是**单条消息回复耗时过长**的问题，不替代 RabbitMQ + Outbox 异步架构。异步架构解决入口并发和任务可靠性，本设计解决后台真正执行一条消息时的推理耗时。

## 当前问题

当前系统存在以下性能问题：

1. **ReAct 是默认路径**
   - 状态查询、规则问答、设定问答、简单剧情推进都可能进入完整 ReAct
   - 很多请求本来只需要一次上下文加载和一次回复生成

2. **LLM 调用次数过多**
   - ReAct 每一轮通常都要等待一次模型输出
   - 总耗时近似等于 `step_count * llm_latency`

3. **上下文获取分散**
   - 模型可能多轮决定是否读取 memory、game state、encounter、rules、lore
   - 上下文拼装依赖模型规划，容易浪费 step

4. **模型层级没有分离**
   - 简单请求和复杂请求使用同样的模型路径
   - 简单问题没有利用更快的 flash 模型

5. **缺少可量化的延迟诊断**
   - 当前不容易直接对比 route 类型、模型类型、step 数和耗时之间的关系

## 范围

本设计包含：

- 请求意图与复杂度路由
- fast path 与 ReAct path 分离
- `fast/pro` 模型分流
- 高层上下文构建
- ReAct 最大轮数限制
- fast path fallback 到 pro ReAct
- 延迟日志与评测指标

本设计不包含：

- 更换具体模型供应商
- 新增 SSE / WebSocket 流式输出
- RabbitMQ / Outbox 可靠性改造
- 前端 UI 大改
- 独立工作流引擎或 step 级 durable execution

## 总体架构

新的 Agent 执行链路为：

```text
用户消息
  -> RouteDecision
  -> ContextBuilder
  -> ModelSelector
  -> Fast Path 或 ReAct Path
  -> Response
```

两条执行路径：

```text
Fast Path:
  高层上下文 -> flash 模型 -> 单次回复

ReAct Path:
  高层上下文 -> pro 模型 -> ReAct(max_steps=2~3) -> 回复
```

核心原则：

- ReAct 不再作为默认路径，只作为复杂请求和不确定请求的处理路径
- 高置信简单请求使用 fast path
- 不确定请求保守走 pro ReAct
- 复杂请求仍可使用工具，但必须限制最大 step 数
- 所有路径都必须记录 route、模型、step 和耗时

## 路由模型

### RouteType

建议定义以下路由类型：

```go
type RouteType string

const (
	RouteStatusQuery       RouteType = "status_query"
	RouteRulesQuery        RouteType = "rules_query"
	RouteLoreQuery         RouteType = "lore_query"
	RouteSimpleSceneAction RouteType = "simple_scene_action"
	RouteCombatAction      RouteType = "combat_action"
	RouteComplexAction     RouteType = "complex_action"
	RouteUnknown           RouteType = "unknown"
)
```

### Complexity

```go
type Complexity string

const (
	ComplexitySimple  Complexity = "simple"
	ComplexityComplex Complexity = "complex"
)
```

### ModelTier

```go
type ModelTier string

const (
	ModelTierFast ModelTier = "fast"
	ModelTierPro  ModelTier = "pro"
)
```

### RouteDecision

```go
type RouteDecision struct {
	RouteType  RouteType
	Complexity Complexity
	ModelTier  ModelTier
	UseReAct   bool
	MaxSteps   int
	Confidence float64
	Reason     string
}
```

## 复杂度分流规则

第一版使用确定性规则，不额外调用 LLM 分类。这样不会为了分类多增加一次模型延迟。

### Fast Path 请求

以下请求默认判定为 `simple / fast / no ReAct`。

#### 状态查询

示例：

- `我现在在哪`
- `我还有多少血`
- `背包里有什么`
- `当前任务是什么`
- `角色状态`
- `当前目标`

输出：

```text
route_type = status_query
complexity = simple
model_tier = fast
use_react = false
max_steps = 0
```

#### 简单规则问答

示例：

- `敏捷影响什么能力`
- `高等精灵适合法师吗`
- `法师有什么特点`
- `这个规则是什么意思`

输出：

```text
route_type = rules_query
complexity = simple
model_tier = fast
use_react = false
max_steps = 0
```

#### 简单设定问答

示例：

- `无光社会是什么`
- `这个城市有什么特征`
- `这个 NPC 是谁`
- `某个地区在哪里`

输出：

```text
route_type = lore_query
complexity = simple
model_tier = fast
use_react = false
max_steps = 0
```

#### 简单剧情推进

示例：

- `继续前进`
- `观察房间`
- `打开门`
- `和他交谈`
- `调查桌子`

输出：

```text
route_type = simple_scene_action
complexity = simple
model_tier = fast
use_react = false
max_steps = 0
```

### ReAct Path 请求

以下请求默认判定为 `complex / pro / ReAct`。

#### 战斗与结算

命中以下语义时升级为复杂请求：

- 攻击
- 伤害
- 命中
- 扣血
- 施法
- 豁免
- 先攻
- 回合
- 战斗
- 治疗
- 死亡
- AC / 护甲

输出：

```text
route_type = combat_action
complexity = complex
model_tier = pro
use_react = true
max_steps = 3
```

#### 多步骤动作

命中以下结构时升级为复杂请求：

- `然后`
- `同时`
- `接着`
- `如果`
- `分别`
- `先...再...`
- `一边...一边...`

输出：

```text
route_type = complex_action
complexity = complex
model_tier = pro
use_react = true
max_steps = 2
```

#### 不确定请求

无法高置信识别为简单请求时，默认走复杂路径：

```text
route_type = unknown
complexity = complex
model_tier = pro
use_react = true
max_steps = 2
```

这样做的原因是：误把复杂请求降级为 fast 会降低回复质量，而误把简单请求升级为 pro 只会带来额外延迟。第一版应优先保证质量。

## 会话状态参与分流

只看用户文本不够。路由器还需要考虑当前 session 状态。

示例：

```text
用户说：“继续”
```

如果当前没有 active encounter：

```text
simple_scene_action -> fast
```

如果当前有 active encounter：

```text
combat_action -> pro ReAct
```

建议路由输入包含：

```go
type RouteInput struct {
	Message          string
	HasActiveCombat  bool
	HasGameState     bool
	RecentToolErrors int
}
```

## 高层上下文构建

新增或强化统一上下文构建能力：

```text
get_agent_context(session_id, user_message)
```

这个能力可以先作为内部 `ContextBuilder` 实现，后续再暴露为 Agent 高层工具。

上下文包应包含：

- 最近消息
- session memory
- game state
- encounter
- 当前目标
- 角色摘要
- rules topK
- lore topK

设计要求：

- fast path 在调用 flash 前必须先获得上下文包
- ReAct path 在进入 pro ReAct 前也先获得上下文包
- ReAct 不再依赖多轮低层工具调用来拼基础上下文
- rules / lore 检索只在对应 route 或 query 命中时执行，避免无意义 RAG

## 模型分流

新增模型分层配置：

```env
FAST_MODEL_PROVIDER=
FAST_MODEL_NAME=
PRO_MODEL_PROVIDER=
PRO_MODEL_NAME=
```

执行规则：

| 路径 | 模型层级 | 调用方式 |
| --- | --- | --- |
| Fast Path | `fast` | flash 单次生成 |
| ReAct Path | `pro` | pro ReAct，最多 2~3 步 |

代码层不应硬编码具体模型名，只依赖 `ModelTier`。实际 provider / model 通过配置加载。

## Fast Path 设计

### status_query

流程：

```text
读取 game_state / encounter / memory
-> 直接格式化回复
-> 可选 flash 润色
```

目标延迟：

```text
0.5s~2s
```

### rules_query

流程：

```text
rules RAG topK
-> flash 单次生成
```

目标延迟：

```text
2s~6s
```

### lore_query

流程：

```text
lore RAG topK
-> flash 单次生成
```

目标延迟：

```text
2s~6s
```

### simple_scene_action

流程：

```text
ContextBuilder
-> flash 单次生成剧情推进
```

目标延迟：

```text
4s~8s
```

## ReAct Path 设计

复杂请求继续使用 ReAct，但必须受控。

建议第一版配置：

```text
normal complex: max_steps = 2
combat/high-risk: max_steps = 3
unknown: max_steps = 2
```

超过最大步数时：

- 汇总已有上下文和工具结果
- 生成当前最合理回复
- 不继续空转
- 记录 `fallback_used = true`

## Fallback 策略

fast path 失败时允许 fallback 到 pro ReAct，但必须有限制。

允许 fallback 的情况：

- 必要上下文缺失
- RAG 没有足够结果
- flash 生成失败
- fast path 识别到任务实际需要状态变更

不允许无限 fallback。一次请求最多触发一次 fast-to-pro fallback。

fallback 后记录：

```text
fallback_used = true
fallback_reason
original_route_type
final_route_type
```

## 观测指标

每次 Agent 执行必须记录：

```text
route_type
complexity
model_tier
use_react
max_steps
actual_steps
intent_route_ms
context_build_ms
rag_ms
llm_call_ms
tool_call_ms
run_total_ms
fallback_used
fallback_reason
```

核心评估指标：

- 平均响应时间
- P95 响应时间
- 平均 ReAct step 数
- LLM 调用次数
- fast path 命中率
- fallback 率
- 成功率

## 预期效果

| 场景 | 当前 | 目标 |
| --- | ---: | ---: |
| 状态查询 | `20s~60s` | `0.5s~2s` |
| 简单规则问答 | `20s~60s` | `2s~6s` |
| 简单设定问答 | `20s~60s` | `2s~6s` |
| 普通剧情推进 | `30s~60s` | `4s~12s` |
| 普通复杂动作 | `40s~60s` | `10s~20s` |
| 战斗复杂动作 | `60s+` | `15s~30s` |

## 实施顺序

1. 增加延迟日志和 step 统计，不改变现有行为
2. 实现 `RouteDecision`、`RouteType`、`Complexity`、`ModelTier`
3. 实现确定性 `Intent / Complexity Router`
4. 增加 `FAST_MODEL_*` 和 `PRO_MODEL_*` 配置
5. 实现 `ModelSelector`
6. 实现 `ContextBuilder / get_agent_context`
7. 实现 `status_query` fast path
8. 实现 `rules_query` fast path
9. 实现 `lore_query` fast path
10. 实现 `simple_scene_action` fast path
11. 将复杂请求接入 pro ReAct
12. 限制 ReAct `max_steps`
13. 增加 fallback 日志
14. 使用 `soak_eval` 对比优化前后结果

## 验收标准

功能标准：

- 简单状态查询不进入 ReAct
- 规则问答和设定问答能走 flash 单次生成
- 复杂战斗请求仍走 pro ReAct
- 不确定请求默认走 pro ReAct
- 当前有 active encounter 时，简单动作可升级到复杂路径
- fast path 失败时能 fallback 到 pro ReAct

性能标准：

- 平均 ReAct step 数明显下降
- fast path 请求 `actual_steps = 0`
- 普通复杂请求 `actual_steps <= 2`
- 战斗复杂请求 `actual_steps <= 3`
- 状态查询目标延迟进入 `0.5s~2s`
- 简单规则/设定问答目标延迟进入 `2s~6s`

评测标准：

- `soak_eval` 记录优化前后平均耗时、P95、成功率、step count
- 成功率不能因为 fast path 明显下降
- 需要输出 fast path 命中率和 fallback 率

## 风险

### 路由误判

风险：

- 复杂请求被错误路由到 fast，导致质量下降

控制：

- 只让高置信简单请求走 fast
- 战斗、多步骤、状态变更、不确定请求保守走 pro
- fast path 可 fallback 到 pro

### flash 幻觉

风险：

- flash 模型在上下文不足时生成不可靠内容

控制：

- flash 必须基于 `ContextBuilder` 输出
- prompt 明确只能依据上下文回答
- 上下文不足时 fallback 到 pro 或明确说明无法确定

### ReAct 轮数过低

风险：

- 复杂任务未完成就被截断

控制：

- 战斗和高风险请求允许 `max_steps = 3`
- 超过 step 后汇总已有工具结果，输出可用回复
- 用 soak eval 检查成功率变化

### 配置复杂

风险：

- fast/pro 模型配置分散，后续难维护

控制：

- 所有模型选择通过 `ModelSelector`
- 业务代码只依赖 `ModelTier`
- 日志记录实际 provider 和 model name

## 非目标

以下能力不在本设计中实现：

- step 级工作流引擎
- 每个 ReAct step 的 durable execution
- SSE token streaming
- UI 展示 token 级流式输出
- 自动学习型路由器

这些可以在延迟优化第一版稳定后再评估。

