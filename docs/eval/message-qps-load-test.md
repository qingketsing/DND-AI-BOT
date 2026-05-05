# 消息接口 QPS 压测

本压测只覆盖消息主链路：

- `POST /sessions/{id}/messages`

目标不是先跑出一个好看的总 QPS，而是回答：

1. 不同并发下消息接口的真实吞吐是多少
2. `P50 / P95 / P99` 如何变化
3. 从哪个并发开始错误率和超时率显著上升
4. 当前瓶颈更偏向模型调用、工具链、还是状态更新

## 脚本

压测脚本：

- [scripts/load/k6-messages.js](/home/qingke/DND-AI-BOT/scripts/load/k6-messages.js)

环境变量模板：

- [scripts/load/.env.loadtest.example](/home/qingke/DND-AI-BOT/scripts/load/.env.loadtest.example)

## 设计原则

### 1. 每个 VU 独立登录、独立建会话

脚本不会让多个并发用户共用同一个 session。

原因：

- 共用 session 会把业务状态冲突带入结果
- 那样测到的是“同会话并发写入冲突”，不是消息接口本身的吞吐

当前脚本中，每个 VU 会在第一次迭代时：

1. `POST /auth/login`
2. `POST /sessions`
3. 缓存自己的 `session_id`
4. 后续只压自己的 `POST /sessions/{id}/messages`

### 2. 固定消息模板

为了让不同轮次可比，脚本不使用随机长对话，而是固定三类消息集：

- `L1`
  - `我们说到哪了`
  - `介绍一下当前设定`
  - `当前的目标是什么`

- `L2`
  - `我检查房间并观察周围`
  - `创建一个高等精灵法师，使用标准数组`
  - `我想检查一下门后有没有动静`

- `L3`
  - `继续攻击并结算伤害`
  - `根据当前 encounter 推进战斗并说明下一步`
  - `我进行一次战斗动作，并按当前状态完整结算`

每个场景只测试一种复杂度，不要混在一起。

## 环境变量

必须提供：

```bash
BASE_URL=http://localhost:8080
EMAIL=loadtest@example.com
PASSWORD=Password123!
```

可选：

```bash
CHANNEL=web
MESSAGE_MODE=L1
STAGES=1:3m,2:3m,5:3m,10:3m
THINK_TIME_SECONDS=1
REQUEST_TIMEOUT=180s
```

### 字段说明

- `MESSAGE_MODE`
  - `L1` / `L2` / `L3`
- `STAGES`
  - 语法：`并发数:持续时间,并发数:持续时间`
  - 例如：`1:3m,2:3m,5:3m`
- `THINK_TIME_SECONDS`
  - 同一个 VU 两次发消息之间的停顿
- `REQUEST_TIMEOUT`
  - 单次消息请求的超时设置

## 运行方式

### 轻消息压测

```bash
set -a
source scripts/load/.env.loadtest.example
set +a

MESSAGE_MODE=L1 \
k6 run scripts/load/k6-messages.js
```

### 中消息压测

```bash
set -a
source scripts/load/.env.loadtest.example
set +a

MESSAGE_MODE=L2 \
k6 run scripts/load/k6-messages.js
```

### 重消息压测

```bash
set -a
source scripts/load/.env.loadtest.example
set +a

MESSAGE_MODE=L3 \
k6 run scripts/load/k6-messages.js
```

### 自定义并发梯度

```bash
set -a
source scripts/load/.env.loadtest.example
set +a

MESSAGE_MODE=L1 \
STAGES='1:2m,2:2m,3:2m,5:2m' \
k6 run scripts/load/k6-messages.js
```

## 推荐压测顺序

不要直接从高并发开始。

建议顺序：

1. `L1`：`1 / 2 / 5`
2. `L2`：`1 / 2 / 5`
3. `L3`：`1 / 2 / 3`
4. 如果前三组都稳定，再尝试 `10` 并发

原因很直接：

- 当前消息主链路延迟高
- 高并发会很快把模型调用和状态更新链路打满
- 先看低并发的稳定区间更有分析价值

## 关注指标

### k6 侧

重点看：

- `http_req_duration`
- `http_req_failed`
- `message_duration_ms`
- `message_failure_rate`
- `iterations`
- `vus`

当前脚本内建阈值：

- `http_req_failed < 1%`
- `message_failure_rate < 1%`
- `message_duration_ms p(95) < 120000`
- `message_duration_ms p(99) < 180000`

这些阈值不是上线标准，只是为了快速识别彻底失稳。

### 服务日志侧

压测期间建议同时观察：

- `http request duration_ms`
- `runtime model call completed duration_ms`
- `agent latency breakdown`
- `tool execution failed`
- `agent run fallback`

如果 `P95` 上升但 `model call duration` 不明显变化，问题更可能在：

- step 数变多
- 工具失败重试
- 上下文过长

如果 `model call duration` 本身暴涨，则更可能是：

- 外部模型限流
- 模型服务排队

## 如何解读结果

### 情况 A：QPS 很低，但错误率低

这通常说明：

- 系统没崩
- 但单请求延迟过高

当前项目更接近这种情况。

### 情况 B：并发一升高，P95 和错误率一起涨

这通常说明：

- 已经进入外部模型或应用内部排队区

### 情况 C：L1 还好，L3 很差

这通常说明：

- 基础链路还能跑
- 真正瓶颈在多 step、工具调用、战斗状态推进

## 当前建议

第一轮只做：

- `MESSAGE_MODE=L1`
- `STAGES=1:3m,2:3m,5:3m`

跑完之后，再决定是否继续上 `L2 / L3`。

这是最稳的起点，也最容易和你现在已有的消息延迟观测对齐。
