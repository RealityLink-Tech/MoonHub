# Context Compactor 上下文压缩器实现文档

**包内文档（配置与阅读顺序）**：[pkg/compactor/docs/README.md](../../pkg/compactor/docs/README.md) · [CONFIG.md](../../pkg/compactor/docs/CONFIG.md)

## 概述

Context Compactor 是一个为 MoonHub 设计的 4 层上下文压缩管道。它通过以下 4 层管道提供智能的上下文管理：

1. **Layer 1: 基于规则的预压缩** - 9 个确定性规则
2. **Layer 2: 消息去重** - Shingle 哈希 + Jaccard 相似度
3. **Layer 3: LLM 摘要生成** - 单次 LLM 调用，带 Token 预算
4. **Layer 4: 分层摘要** - L0(200)、L1(1000)、L2(3000) Token

## 实现状态

### ✅ 已完成

| 组件 | 状态 | 说明 |
|------|------|------|
| 核心实现 | ✅ 完成 | 7 个核心文件全部实现 |
| 配置集成 | ✅ 完成 | defaults.go 已添加默认配置 |
| Agent 集成 | ✅ 完成 | instance.go, loop.go 已集成 |
| 分层摘要 | ✅ 完成 | 根据上下文压力动态选择层级 |
| 测试文件 | ✅ 完成 | 5 个测试文件覆盖核心功能 |

## 包结构

```
pkg/compactor/
├── compactor.go        # 主编排器（CompactorEngine 接口）
├── config.go           # 配置类型和默认值
├── config_test.go      # 配置测试
├── dedup.go            # Layer 2: Shingle 哈希 + Jaccard 相似度
├── dedup_test.go       # 去重测试
├── rules.go            # Layer 1: 9 个确定性预压缩规则
├── rules_test.go       # 规则测试
├── store.go            # SQLite 存储分层摘要
├── tiers.go            # Layer 4: L0/L1/L2 分层生成
├── tiers_test.go       # 分层测试
├── tokens.go           # Token 估算工具
└── tokens_test.go      # Token 估算测试
```

## 架构

### 4 层管道

```
消息 → [Layer 1: 规则] → [Layer 2: 去重] → [Layer 3: LLM 摘要] → [Layer 4: 分层]
```

| 层级 | 功能 | 输入 | 输出 |
|------|----------|-------|--------|
| 1 | 基于规则的预压缩 | 原始消息 | 压缩后的消息 |
| 2 | 消息去重 | 压缩后的消息 | 唯一消息 |
| 3 | LLM 摘要生成 | 唯一消息 | L2 摘要（3000 Token） |
| 4 | 分层摘要 | L2 摘要 | L0(200) + L1(1000) + L2(3000) |

### Layer 1: 预压缩规则（9 个规则）

| 规则 | 功能 | 节省 |
|------|----------|---------|
| CJK 标点规范化 | 将全角标点转换为 ASCII | ~1 Token/替换 |
| 折叠空白 | 删除尾随空格，折叠空行 | 可变 |
| 行去重 | 删除完全重复的行 | 可变 |
| 删除空章节 | 消除空的 Markdown 章节 | 可变 |
| 压缩表格 | 将表格转换为 key:value 格式 | 30-50% |
| 删除 Emoji | 删除 Emoji 字符 | 可变 |
| 合并相似项目符号 | 使用二元相似度合并项目符号 | 可变 |
| 合并短项目符号 | 合并连续的短项目符号 | 可变 |
| 删除装饰线 | 删除 "---"、"***"、"===" | 可变 |

### Layer 2: 消息去重

使用 Shingle 哈希（词级别 3-gram）和 Jaccard 相似度检测近似重复的消息：

- 默认相似度阈值：0.6
- 在重复对中删除较早的消息（保留较新的）
- 支持大消息集的并行处理

### Layer 3: LLM 摘要生成

单次 LLM 调用，使用结构化提示生成综合摘要：

- Token 预算：3000 Token（可配置）
- 保留：用户身份、决策、纠正、任务、偏好
- 结构化输出，包含清晰的章节

### Layer 4: 分层摘要

基于优先级的分层生成：

| 章节 | 优先级 |
|---------|----------|
| 身份/用户信息 | 10 |
| 决策/纠正 | 9 |
| 活跃任务/行动 | 8 |
| 偏好 | 7 |
| 上下文/主题 | 4-5 |
| 备注 | 3 |

**分层选择策略：**

| 历史长度 | 使用层级 | 说明 |
|----------|----------|------|
| ≤ 20 条消息 | L2 (3000 tokens) | 完整摘要 |
| 21-40 条消息 | L1 (1000 tokens) | 工作记忆 |
| > 40 条消息 | L0 (200 tokens) | 极限压缩 |

## 配置

### pkg/config/config.go

```go
type CompactorConfig struct {
    Enabled                 bool
    TriggerTokenPercent     int     // 默认值: 70
    KeepRecent              int     // 默认值: 10
    TierBudgets             TierBudgetsConfig
    DedupEnabled            bool
    DedupSimilarityThreshold float64  // 默认值: 0.6
    StripEmoji              bool
    RemoveDuplicateLines    bool
    NormalizeCJK            bool
    SmartRuleSelection      bool
    ParallelProcessing      bool
    IncrementalCompaction   bool
    SummarizationModel      string  // 可选
}

type TierBudgetsConfig struct {
    L0 int  // 默认值: 200
    L1 int  // 默认值: 1000
    L2 int  // 默认值: 3000
}
```

### config.json 示例

```json
{
  "compactor": {
    "enabled": true,
    "trigger_token_percent": 70,
    "keep_recent": 10,
    "tier_budgets": {
      "l0": 200,
      "l1": 1000,
      "l2": 3000
    },
    "dedup_enabled": true,
    "dedup_similarity_threshold": 0.6,
    "strip_emoji": true,
    "remove_duplicate_lines": true,
    "normalize_cjk": true,
    "smart_rule_selection": true,
    "parallel_processing": true,
    "incremental_compaction": true
  }
}
```

## 集成点

### 1. AgentInstance (pkg/agent/instance.go)

✅ **已实现** - 在 `AgentInstance` 结构体中添加了 `Compactor` 字段：

```go
type AgentInstance struct {
    // ... 现有字段 ...
    Compactor                    compactor.CompactorEngine
    CompactorTriggerTokenPercent int // 与 compactor 内 ShouldTrigger 一致，用于 maybeCompact 门槛
}
```

在 `NewAgentInstance()` 中初始化：

```go
// 如果启用则初始化压缩器
var compactorEngine compactor.CompactorEngine
if cfg.Compactor.Enabled && provider != nil {
    compactorCfg := compactor.Config{
        Enabled:                  cfg.Compactor.Enabled, // 必须传入，否则 CompactIfNeeded 会因零值 false 直接返回
        DBPath:                   filepath.Join(workspace, "compactor.db"),
        TriggerTokenPercent:      cfg.Compactor.TriggerTokenPercent,
        KeepRecent:               cfg.Compactor.KeepRecent,
        TierBudgets: compactor.TierBudgets{
            L0: cfg.Compactor.TierBudgets.L0,
            L1: cfg.Compactor.TierBudgets.L1,
            L2: cfg.Compactor.TierBudgets.L2,
        },
        // ... 其他配置 ...
    }
    compactorEngine, err = compactor.New(compactorCfg, provider)
    if err != nil {
        log.Printf("compactor: 初始化失败: %v; 使用旧版摘要", err)
    }
}
```

### 2. Agent Loop (pkg/agent/loop.go)

✅ **已实现** - 新增 `maybeCompact()` 和 `forceCompact()` 函数：

```go
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey, channel, chatID string) {
    // 使用 compactor 如果可用
    if agent.Compactor != nil {
        al.maybeCompact(agent, sessionKey, channel, chatID)
        return
    }
    // 旧版摘要逻辑...
}

func (al *AgentLoop) maybeCompact(agent *AgentInstance, sessionKey, channel, chatID string) {
    // 基于 Token 计数的动态阈值（与 compactor.ShouldTrigger 使用同一配置：CompactorTriggerTokenPercent）
    history := agent.Sessions.GetHistory(sessionKey)
    tokenEstimate := agent.Compactor.EstimateTokensFromMessages(history)
    threshold := int(float64(agent.ContextWindow) * float64(agent.CompactorTriggerTokenPercent) / 100)

    if tokenEstimate > threshold {
        // 异步执行压缩（goroutine 内会重新 GetHistory，避免用过期快照更新 session）
    }
}
```

### 3. Agent Loop - 分层摘要支持 (pkg/agent/loop.go)

✅ **已实现** - 新增 `getTieredSummary()` 函数：

```go
func (al *AgentLoop) getTieredSummary(ctx context.Context, agent *AgentInstance, sessionKey string, historyLen int) string {
    if agent.Compactor == nil {
        return ""
    }

    tier := compactor.SelectTier(historyLen)
    tieredSummary, err := agent.Compactor.GetTieredSummary(ctx, sessionKey)
    // return tieredSummary.GetTier(tier)
}
```

## 存储架构

SQLite 数据库：`~/.moonhub/workspace/compactor.db`

### 表结构

```sql
-- 分层摘要存储
CREATE TABLE tiered_summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_key TEXT NOT NULL UNIQUE,
    l0_summary TEXT NOT NULL,
    l1_summary TEXT NOT NULL,
    l2_summary TEXT NOT NULL,
    messages_before INTEGER NOT NULL,
    messages_summarized INTEGER NOT NULL,
    tokens_before INTEGER NOT NULL,
    tokens_after INTEGER NOT NULL,
    compression_ratio REAL NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- 增量压缩状态
CREATE TABLE compaction_state (
    session_key TEXT PRIMARY KEY,
    last_compaction_timestamp INTEGER NOT NULL,
    last_message_index INTEGER NOT NULL,
    total_compactions INTEGER NOT NULL DEFAULT 0
);

-- 压缩历史记录（用于分析）
CREATE TABLE compaction_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_key TEXT NOT NULL,
    compaction_id INTEGER NOT NULL,
    messages_removed INTEGER NOT NULL,
    dedup_groups INTEGER NOT NULL,
    rules_applied TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (compaction_id) REFERENCES tiered_summaries(id)
);
```

## 优化特性

### 1. 并行处理

- Layer 1: 使用 goroutine 并行处理消息
- Layer 4: 从 L2 并行生成 L0 和 L1 分层

### 2. 增量压缩

- 在 `compaction_state` 表中跟踪上次压缩索引
- 只处理上次压缩后的新消息
- 将新摘要与现有 L2 摘要合并

### 3. 智能规则选择

- 检测 CJK 内容 → 启用 CJK 规范化
- 检测 Markdown 表格 → 启用表格压缩
- 检测代码块 → 跳过项目符号合并
- 在应用规则前分析内容

## 使用方式

压缩器在以下情况自动触发：
- Token 使用量超过上下文窗口的 `trigger_token_percent`
- 压缩在后台异步运行
- 分层摘要持久化存储在 SQLite 中

### 禁用压缩器

在 `config.json` 中设置：

```json
{
  "compactor": {
    "enabled": false
  }
}
```

## 测试

```bash
# 运行单元测试
go test ./pkg/compactor/... -v

# 运行覆盖率测试
go test ./pkg/compactor/... -cover

# 运行基准测试
go test ./pkg/compactor/... -bench=. -benchmem
```

### 测试覆盖

| 测试文件 | 覆盖内容 |
|---------|---------|
| `tokens_test.go` | Token 估算、CJK 检测、截断 |
| `rules_test.go` | 9 个预压缩规则、智能规则选择 |
| `dedup_test.go` | Shingle 哈希、Jaccard 相似度、去重 |
| `config_test.go` | 配置验证、默认值、触发条件 |
| `tiers_test.go` | 分层生成、优先级评分、层级选择 |

## 参考

技术特点：
1. 纯 Go 实现
2. 动态基于 Token 的触发（而非固定消息计数）
3. 三种优化全部实现：并行处理、增量压缩、智能规则选择
4. SQLite 持久化存储所有分层摘要

## 实现文件清单

### 核心实现文件

| 文件路径 | 说明 |
|---------|------|
| `pkg/compactor/tokens.go` | Token 估算工具，支持 CJK 字符 |
| `pkg/compactor/rules.go` | 9 个预压缩规则实现 |
| `pkg/compactor/dedup.go` | Shingle 哈希和 Jaccard 相似度去重 |
| `pkg/compactor/tiers.go` | L0/L1/L2 分层摘要生成 |
| `pkg/compactor/store.go` | SQLite 存储实现 |
| `pkg/compactor/config.go` | 配置类型定义 |
| `pkg/compactor/compactor.go` | 主编排器和 CompactorEngine 接口 |

### 测试文件

| 文件路径 | 说明 |
|---------|------|
| `pkg/compactor/tokens_test.go` | Token 估算测试 |
| `pkg/compactor/rules_test.go` | 预压缩规则测试 |
| `pkg/compactor/dedup_test.go` | 消息去重测试 |
| `pkg/compactor/config_test.go` | 配置测试 |
| `pkg/compactor/tiers_test.go` | 分层摘要测试 |

### 集成修改

| 文件路径 | 修改内容 |
|---------|---------|
| `pkg/config/config.go` | 添加 CompactorConfig 和 TierBudgetsConfig |
| `pkg/config/defaults.go` | 添加 Compactor 默认配置 |
| `pkg/agent/instance.go` | 添加 Compactor 字段和初始化 |
| `pkg/agent/loop.go` | 集成 maybeCompact, forceCompact, getTieredSummary |
