# RAG 检索误差分析

日期：2026-05-04

数据来源：
- [reports/eval/rag_eval_report.json](/home/qingke/DND-AI-BOT/reports/eval/rag_eval_report.json)
- [reports/eval/rag_eval_report.md](/home/qingke/DND-AI-BOT/reports/eval/rag_eval_report.md)

## 分析范围

本次分析使用当前已经人工确认的 benchmark 子集：

- query 总数：46
- rules：21
- lore：25

目标不是重复罗列 Recall@K 数字，而是回答两件事：

1. 为什么 `lexical` 表现明显偏差
2. `hybrid` 还剩下哪些真实瓶颈

## 结果概览

### 总体结果

| 检索后端 | MRR | Recall@1 | Recall@3 | Recall@5 | Recall@10 |
| --- | ---: | ---: | ---: | ---: | ---: |
| hybrid | 0.8826 | 0.6178 | 0.8098 | 0.9094 | 0.9783 |
| lexical | 0.3696 | 0.3116 | 0.3587 | 0.3587 | 0.3587 |

### 按知识库划分

| 知识库 | 检索后端 | MRR | Recall@1 | Recall@10 |
| --- | --- | ---: | ---: | ---: |
| rules | hybrid | 0.7429 | 0.4167 | 0.9524 |
| rules | lexical | 0.0476 | 0.0476 | 0.0476 |
| lore | hybrid | 1.0000 | 0.7867 | 1.0000 |
| lore | lexical | 0.6400 | 0.5333 | 0.6200 |

### 按 query 类型划分

| query 类型 | 检索后端 | Recall@1 | Recall@10 | MRR |
| --- | --- | ---: | ---: | ---: |
| exact_name | hybrid | 0.8500 | 1.0000 | 1.0000 |
| exact_name | lexical | 0.6000 | 0.6000 | 0.6000 |
| semantic | hybrid | 0.6377 | 1.0000 | 0.8667 |
| semantic | lexical | 0.2826 | 0.3478 | 0.3478 |
| alias | hybrid | 0.4048 | 0.8571 | 0.7381 |
| alias | lexical | 0.1429 | 0.1429 | 0.1429 |
| multi_chunk | hybrid | 0.4028 | 1.0000 | 0.9167 |
| multi_chunk | lexical | 0.1389 | 0.2500 | 0.3333 |

## 主要结论

### 1. lexical 的核心问题不是排序，而是召回失败

最明显的信号是：

- Recall@3 = 0.3587
- Recall@5 = 0.3587
- Recall@10 = 0.3587

这说明 `lexical` 的问题不是“正确 chunk 召回到了，但是排得太后面”。
更真实的情况是：

- 大量 query 在 top10 里根本没有正确 chunk

如果只是排序不佳，通常会看到：

- Recall@1 较低
- 但 Recall@3 / @5 / @10 会持续上升

当前结果不是这个形态。

### 2. lexical 在 rules 知识库上几乎不可用

具体拆分如下：

- `lexical + rules`：21 条 query
  - 空结果：20
  - 命中：1
  - 有结果但 miss：0

- `lexical + lore`：25 条 query
  - 空结果：8
  - 命中：16
  - 有结果但 miss：1

这说明问题不只是“hybrid 更强”，而是：

- 在 `rules` 上，当前 lexical 基本拿不到可用结果
- 在 `lore` 上，lexical 仍然保留部分价值，尤其是专有名词和直接设定名词查询

因此后续不能再把 `rules` 和 `lore` 当成同一种检索问题处理。

### 3. hybrid 已经解决召回问题，但 top1 排序仍然不稳定

以下 query 都属于：

- `hybrid` 能命中
- 但第一个相关 chunk 不在 rank1

| query_id | query | 第一个命中位置 |
| --- | --- | ---: |
| rules-001 | 创建角色时第一步要做什么 | 5 |
| rules-012 | 敏捷属性影响什么能力 | 5 |
| rules-016 | 魅力属性和哪些职业最相关 | 2 |
| rules-017 | 高等精灵适合法师吗 | 6 |
| rules-020 | 法师这个职业有什么特点 | 5 |
| rules-024 | 个性与背景章节是做什么的 | 3 |

这类 query 的共同特征是：

- 章节概括型问题
- 属性到能力映射的问题
- “适合 / 相关 / 区别 / 主要讲什么” 这类解释型表述

它们通常不是简单找一个精确术语，而是要在一组邻近 chunk 里找“最能回答问题的那块”。
所以现在 hybrid 的主要问题已经从“能不能召回”变成“能不能排第一”。

### 4. hybrid 仍然 miss 了一条 alias 较重的规则 query

当前 `hybrid` 在 rules 上唯一一条 miss：

- `rules-009`：`27 点购点法是什么`

目标 chunk：

- `rules:phb:chapter:01-character-creation:0017`

但 hybrid 的 top10 没有召回这条。

这说明这里更像是：

- alias / 同义表达归一化问题

用户写法是：

- `27 点购点法`

而正文更可能写成：

- 正式规则名称
- 或一段描述性说明

这类 query 靠现有检索直接命中并不稳定。

## 代表性误差样本

### lexical miss，hybrid hit

这些样本最值得后续拿来做针对性调优。

1. `rules-001`：`创建角色时第一步要做什么`
   - lexical：空结果
   - hybrid：rank 5 命中
   - 问题类型：章节步骤型语义查询

2. `rules-012`：`敏捷属性影响什么能力`
   - lexical：空结果
   - hybrid：rank 5 命中
   - 问题类型：属性含义分散在多个规则块中

3. `rules-017`：`高等精灵适合法师吗`
   - lexical：空结果
   - hybrid：rank 6 命中
   - 问题类型：用户问法是“适合性判断”，正文并不一定包含“适合”这个词

4. `lore-019`：`滑门是做什么的`
   - lexical：错误召回 `无光社会:0003`
   - hybrid：rank 1 直接命中 `gates-to-the-city:0001`
   - 问题类型：译名 / 别名不稳定

5. `lore-023`：`大厅和走廊有什么共同特征`
   - lexical：空结果
   - hybrid：两个相关 chunk 都在前列
   - 问题类型：跨 chunk 的设定概括问题

## 可能的根因

### A. rules 知识库的 lexical 路径对中文检索不友好

`rules` 上的失败太极端，不太可能只是普通排序波动。
当前 lexical 路径很可能存在以下一个或多个问题：

- 中文分词对长规则正文不友好
- 用户问法和正文用词重叠度低
- 标题 / 节标题没有被单独强化
- 没有别名和术语归一化

### B. 很多 query 本质上是语义问题，不是字面匹配问题

典型例子：

- `适合法师吗`
- `主要讲什么`
- `有什么区别`
- `和哪些职业最相关`

这类 query 需要的是解释、归纳、映射，而不是简单关键词重合。
这也是为什么 dense retrieval 明显优于 lexical。

### C. 数字型 alias 和口语化表述没有归一化

例如：

- `27 点购点法`

如果不把这类 query 改写到更接近正文的表述上，单纯依赖 embedding 或 FTS 都会有不稳定性。

### D. hybrid 缺少最后一层轻量 rerank

现在 hybrid 已经能把正确 chunk 找到附近，但还不一定放到第一位。
这通常说明：

- title 权重不够
- heading 权重不够
- 没有 query type 感知的融合权重
- 没有一个轻量的最后排序阶段

## 后续优化建议

### 优先级 1：先检查 rules 知识库的 lexical 建索引方式

不要一上来就做 prompt 或模型层补丁。
先确认底层检索路径是否正常。

重点检查：

- `rules` chunk 在 FTS 中是如何被分词的
- 标题 / 节标题是否单独建索引
- 中文 query 是否做了规范化处理
- `rules` 和 `lore` 是否错误地共用了一套 lexical 策略

预期目标：

- 判断当前 `rules lexical` 是“完全坏掉”还是“只是参数没调好”

### 优先级 2：补一层轻量 query rewrite / alias normalization

优先处理以下模式：

- 数字型 alias：`27 点购点法`
- 适配型表述：`适合法师吗`
- 章节概括型表述：`主要讲什么`
- 对比型表述：`有什么区别`

第一阶段不需要 LLM rewrite。
先做确定性的 rewrite 词表就够了，用来验证收益。

### 优先级 3：提升 hybrid 的 top1 排序质量

目标不是继续提高 Recall@10，而是提高：

- Recall@1
- MRR

建议先做轻量方案：

- title boost
- heading boost
- 对 `rules` 的 query-type-aware 融合权重

不建议第一步就上重 reranker。

### 优先级 4：分开调优 rules 和 lore

当前证据已经足够说明：

- `lore` 上 lexical 仍然有一定价值
- `rules` 上 lexical 基本失效

因此：

- `rules` 应更偏向语义检索
- `lore` 可以继续保留 lexical 的辅助作用

统一策略会浪费两边的最优空间。

## 可执行的后续任务

1. 做一个 `lexical miss / hybrid hit` 的样本导出脚本
2. 给 benchmark 增加 query rewrite 钩子，做小范围 A/B
3. 在不改 goldset 的前提下分别测试：
   - lexical 调优
   - alias normalization
   - hybrid 权重调整
4. 每次只改一个变量，再重新跑 Recall@K

## 最终结论

这轮 benchmark 已经足以支持一个明确判断：

- `hybrid` 应该是当前项目的默认检索后端
- `lexical` 只在 `lore exact-name` 类问题上保留有限价值
- 当前下一阶段的主要瓶颈已经不是“能不能召回”，而是：
  - `rules` 的 lexical 建索引质量
  - alias / 术语归一化
  - hybrid 的 top1 排序质量
