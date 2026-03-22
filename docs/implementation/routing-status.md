# Smart Router V2 4级路由 - 实现状态

**源码包**：`pkg/routing/` · **仓库文档索引**：[docs/README.md](../README.md)

## 实现概述

MoonHub 的 Smart Router V2 将原有的 2 级路由 (light/heavy) 升级为 4 级路由系统，同时保留了 MoonHub 现有的语言无关结构化特征、会话历史感知等优势。

## 实现状态摘要

| 功能模块 | 状态 | 文件 | 测试 |
|---------|------|------|------|
| 4 级分类 (QueryTier) | ✅ 完成 | `tier.go` | ✅ |
| V2 分类器 (RuleClassifierV2) | ✅ 完成 | `classifier_v2.go` | ✅ 14 tests |
| V2 路由器 (RouterV2) | ✅ 完成 | `router.go` | ✅ 13 tests |
| 特征提取 (Features) | ✅ 完成 | `features.go` | ✅ |
| 配置支持 | ✅ 完成 | `config/config.go` | ✅ |
| Agent 集成 | ✅ 完成 | `agent/instance.go`, `agent/loop.go` | ✅ |
| Metrics 收集 | ✅ 完成 | `metrics.go` | ✅ 6 tests |
| 决策记录器 | ✅ 完成 | `recorder.go` | ✅ 9 tests |
| HTTP Endpoints | ✅ 完成 | `health/server.go` | ✅ |
| 权重调优 | 📋 待定 | - | - |

**总计**: 42 个单元测试，全部通过

## 已完成的工作

### 1. 核心包结构

```
pkg/routing/
├── tier.go                # QueryTier 类型定义和 DefaultTierBoundaries
├── classifier.go          # 原有 Classifier 接口 + 新增 Signal, ClassificationResult, ClassifierV2
├── classifier_v2.go       # RuleClassifierV2 实现（短消息降权、附件硬门控、Sigmoid 置信度）
├── router.go              # 原有 Router + 新增 TierModelMapping, RouterConfigV2, RouterV2
├── features.go            # Features 特征提取（语言无关）
├── metrics.go             # Metrics 收集器（隐私安全）
├── recorder.go            # 决策记录器（隐私安全）
├── classifier_v2_test.go  # V2 分类器单元测试 (14 个)
├── router_v2_test.go      # V2 路由器单元测试 (13 个)
├── metrics_test.go        # Metrics 单元测试 (6 个)
└── recorder_test.go       # Recorder 单元测试 (9 个)
```

### 2. 核心功能

#### 2.1 4 级分类 (QueryTier)

| Tier | 分数范围 | 说明 | 典型场景 |
|------|---------|------|---------|
| `simple` | [-1.0, -0.05) | 极简任务 | 问候语、确认、简单问答 |
| `moderate` | [-0.05, 0.15) | 中等任务 | 短问题、简单指令 |
| `complex` | [0.15, 0.35) | 复杂任务 | 代码编写、长上下文、工具调用 |
| `reasoning` | [0.35, 1.0] | 推理任务 | 多模态、深度分析、附件处理 |

#### 2.2 特征权重

| 特征 | 权重 | 说明 |
|------|------|------|
| 短消息 (<20 tokens) | -0.10 | 负分，路由到简单模型 |
| 短–中 (20–50 tokens) | +0.06 | 介于问候与中等长度之间 |
| 中等消息 (51–200 tokens) | +0.15 | 中等长度（阈值用 `>50`，恰好 50 token 走短–中档） |
| 长消息 (>200 tokens) | +0.35 | 长文本需要更强模型 |
| 代码块存在 | +0.40 | 编程任务需要重型模型 |
| 工具调用 >3 | +0.25 | 密集工具使用表示复杂工作流 |
| 工具调用 1-3 | +0.10 | 活跃的代理工作流 |
| 对话深度 >10 | +0.10 | 长会话隐含复杂度 |
| 附件存在 | 1.0 (硬门控) | 多模态直接路由到推理模型 |

#### 2.3 置信度计算

使用 Sigmoid 函数基于分数距离边界的距离计算置信度：
- 分数在层级中心：高置信度
- 分数靠近边界：低置信度
- 附件硬门控：100% 置信度

#### 2.4 信号追踪 (Signal)

每个分类结果包含信号追踪，便于调试和可观测性：

```go
type Signal struct {
    Name         string  // 信号名称，如 "short_message", "code_block"
    Contribution float64 // 对分数的贡献值
    RawValue     any     // 原始值，如 token 数量、代码块数量
}
```

### 3. 集成点

#### 3.1 pkg/config/config.go

新增 4 级路由配置：

```go
type RoutingConfig struct {
    Enabled    bool    `json:"enabled"`
    LightModel string  `json:"light_model"`  // 2-tier 模式
    Threshold  float64 `json:"threshold"`    // 2-tier 模式

    // V2 4-tier
    TierMapping    map[string]string      `json:"tier_mapping,omitempty"`
    TierBoundaries *TierBoundariesConfig  `json:"tier_boundaries,omitempty"`
}

type TierBoundariesConfig struct {
    // 指针字段：可区分「未配置」与显式 0.0（例如 simple_moderate: 0）
    SimpleModerate   *float64 `json:"simple_moderate,omitempty"`
    ModerateComplex  *float64 `json:"moderate_complex,omitempty"`
    ComplexReasoning *float64 `json:"complex_reasoning,omitempty"`
}
```

- `tier_mapping` 的 key 必须为 `simple` / `moderate` / `complex` / `reasoning`（大小写敏感）；未知 key 会记录日志并跳过。
- `tier_boundaries` 在任一 cutpoint 被设置时生效；须满足 **sm < mc < cr** 且均在 **[-1, 1]** 内，否则回退默认边界并打日志。
- 健康检查 `GET /routing/decisions?tier=...` 仅接受上述四档；非法值返回 **400**。

#### 3.2 pkg/agent/instance.go

新增字段和初始化逻辑：

```go
type AgentInstance struct {
    // ... 现有字段 ...
    Router         *routing.Router           // 2-tier 路由器
    RouterV2       *routing.RouterV2         // 4-tier 路由器
    TierCandidates map[routing.QueryTier][]providers.FallbackCandidate // 每层预解析的候选
    LightCandidates []providers.FallbackCandidate // 2-tier 兼容
}
```

初始化逻辑：
- 检测 `tier_mapping` 配置 → 启用 4-tier 模式
- 否则检测 `light_model` → 使用 2-tier 模式
- 为每个 tier 预解析 provider candidates

#### 3.3 pkg/agent/loop.go

更新 `selectCandidates` 支持 4 级路由：

```go
func (al *AgentLoop) selectCandidates(
    agent *AgentInstance,
    userMsg string,
    history []providers.Message,
) (candidates []providers.FallbackCandidate, model string) {
    // V2 4-tier routing
    if agent.RouterV2 != nil && agent.RouterV2.IsV2Enabled() && len(agent.TierCandidates) > 0 {
        selectedModel, tier, result := agent.RouterV2.SelectModel(userMsg, history, agent.Model)
        if tierCandidates, ok := agent.TierCandidates[tier]; ok && len(tierCandidates) > 0 {
            logger.InfoCF("agent", "Model routing: tier selected", map[string]any{
                "tier":       tier.String(),
                "model":      selectedModel,
                "score":      result.Score,
                "confidence": result.Confidence,
            })
            return tierCandidates, selectedModel
        }
    }
    // Legacy 2-tier fallback ...
}
```

### 4. 配置示例

#### 4-tier 模式

```json
{
  "agents": {
    "defaults": {
      "routing": {
        "enabled": true,
        "tier_mapping": {
          "simple": "glm-4-flash",
          "moderate": "glm-4-flash",
          "complex": "glm-4-plus",
          "reasoning": "claude-opus-4-5"
        }
      }
    }
  }
}
```

#### 2-tier 模式 (向后兼容)

```json
{
  "agents": {
    "defaults": {
      "routing": {
        "enabled": true,
        "light_model": "glm-4-flash",
        "threshold": 0.35
      }
    }
  }
}
```

#### 自定义边界

```json
{
  "agents": {
    "defaults": {
      "routing": {
        "enabled": true,
        "tier_mapping": {
          "simple": "model-a",
          "moderate": "model-b",
          "complex": "model-c",
          "reasoning": "model-d"
        },
        "tier_boundaries": {
          "simple_moderate": 0.0,
          "moderate_complex": 0.25,
          "complex_reasoning": 0.5
        }
      }
    }
  }
}
```

### 5. 向后兼容性

| 场景 | 行为 |
|------|------|
| 只配置 `light_model` + `threshold` | 使用 2-tier legacy 模式 |
| 配置 `tier_mapping` | 使用 4-tier V2 模式 |
| 都不配置 | 禁用路由，使用 primary model |
| 调用 `SelectModelLegacy` | 自动映射 4-tier → 2-tier |

## 单元测试

### 测试覆盖

| 测试文件 | 测试函数数量 | 覆盖内容 |
|---------|-------------|---------|
| `classifier_v2_test.go` | 14 | 短消息负分、附件硬门控、代码块分类、长消息分类、工具调用分类、对话深度、自定义边界、置信度计算、信号追踪、ScoreToTier 映射、分数范围 |
| `router_v2_test.go` | 13 | 4-tier 模式启用、2-tier 回退、各 tier 路由选择、缺失 tier 映射、Legacy 模式、自定义边界、阈值测试、LightModel 测试、无路由配置、集成测试 |
| `metrics_test.go` | 6 | 基本记录和检索、Reset 功能、并发安全性、NoOp 收集器、全局设置、直方图分布 |
| `recorder_test.go` | 9 | 基本记录、Ring buffer、按 tier 过滤、统计计算、禁用状态、Clear 功能、信号深度限制、隐私保护验证 |

**总计**: 42 个测试函数，覆盖核心功能和边界情况

### 测试结果

```
=== 所有测试通过 ===
ok      github.com/RealityLink-Tech/MoonHub/pkg/routing
```

## 性能目标

- 分类计算: < 1µs 每次消息（纯规则，无 API 调用）
- 内存占用: 零额外分配（除返回结构体外）
- 并发安全: RouterV2 可被多个 goroutine 安全使用

## 设计决策

### 1. 短消息负分设计

- **问题**: 原有 2-tier 系统无法区分"问候语"和"简单问题"
- **解决**: 为 <20 token 的消息添加 -0.10 负分，使其落入 `simple` tier
- **效果**: 问候语如"你好"、"hi"会被路由到最便宜的模型

### 2. 附件硬门控

- **问题**: 多模态消息需要特定模型能力
- **解决**: 检测到附件时直接返回 TierReasoning, Score=1.0, Confidence=1.0
- **效果**: 跳过所有其他计算，保证多模态请求始终使用支持视觉的模型

### 3. 置信度输出

- **问题**: 需要知道路由决策的可信程度
- **解决**: 使用 Sigmoid 函数计算基于分数到边界距离的置信度
- **效果**: 可用于日志、监控和未来可能的 A/B 测试

### 4. 信号追踪

- **问题**: 需要解释为什么某个消息被路由到特定 tier
- **解决**: 每个分类结果包含 Signals 列表，记录所有贡献因素
- **效果**: 便于调试和优化权重

## 文件清单

### 新建文件

| 文件路径 | 说明 |
|---------|------|
| `pkg/routing/tier.go` | QueryTier 类型和 DefaultTierBoundaries |
| `pkg/routing/classifier_v2.go` | RuleClassifierV2 实现 |
| `pkg/routing/classifier_v2_test.go` | V2 分类器单元测试 |
| `pkg/routing/router_v2_test.go` | V2 路由器单元测试 |
| `pkg/routing/metrics.go` | Metrics 收集器实现 |
| `pkg/routing/metrics_test.go` | Metrics 单元测试 |
| `pkg/routing/recorder.go` | 决策记录器实现 |
| `pkg/routing/recorder_test.go` | Recorder 单元测试 |

### 修改文件

| 文件路径 | 修改内容 |
|---------|---------|
| `pkg/routing/classifier.go` | 添加 Signal, ClassificationResult, ClassifierV2 接口 |
| `pkg/routing/router.go` | 添加 TierModelMapping, RouterConfigV2, RouterV2，集成 metrics/recorder |
| `pkg/config/config.go` | 扩展 RoutingConfig 添加 TierMapping 和 TierBoundariesConfig |
| `pkg/agent/instance.go` | 添加 RouterV2, TierCandidates 字段和初始化逻辑 |
| `pkg/agent/loop.go` | 更新 selectCandidates 支持 4-tier 路由 |
| `pkg/health/server.go` | 添加 /metrics, /routing/decisions, /routing/stats endpoints |

## 待完成工作 (TODO)

### 1. 基于反馈的权重调优 (低优先级)
- 收集实际使用数据
- 根据用户反馈微调权重
- 可能需要针对不同语言调整

## 可观测性系统

已实现完整的可观测性系统，**完全保护用户隐私**（不记录消息内容）。

### Metrics 收集 (pkg/routing/metrics.go)

```go
// 全局 metrics 收集器
collector := routing.NewDefaultMetricsCollector()
routing.SetGlobalMetrics(collector)

// 获取快照
snapshot := collector.GetSnapshot()
// snapshot.TotalClassifications - 总分类次数
// snapshot.TierCounts - 各 tier 的计数
// snapshot.AvgScore - 平均分数
// snapshot.AvgConfidence - 平均置信度
// snapshot.SignalCounts - 各信号的触发次数
// snapshot.ScoreDistribution - 分数分布直方图
// snapshot.ConfidenceDistribution - 置信度分布直方图
```

**隐私保护**：只收集聚合统计数据，**不记录用户消息内容**。

### 决策记录器 (pkg/routing/recorder.go)

```go
// 初始化决策记录器
cfg := routing.DefaultDecisionRecorderConfig()
cfg.BufferSize = 1000  // 保留最近 1000 条决策
cfg.MaxSignalDepth = 10  // 每条决策最多记录 10 个信号
recorder := routing.NewDecisionRecorder(cfg)

// 获取最近决策
decisions := recorder.GetRecent(50)

// 按 tier 过滤
simpleDecisions := recorder.GetByTier(routing.TierSimple, 10)

// 获取统计信息
stats := recorder.GetStats()
```

**隐私保护**：
- 不记录原始消息内容
- Signal 的 RawValue 被剥离
- AgentID 应使用哈希/派生值

### HTTP Endpoints (pkg/health/server.go)

| Endpoint | 说明 |
|----------|------|
| `GET /metrics` | 返回聚合的 routing metrics (JSON) |
| `GET /routing/decisions?limit=50&tier=simple` | 返回最近的决策记录 |
| `GET /routing/stats` | 返回聚合统计信息 |

示例响应：

```json
// GET /metrics
{
  "timestamp": "2026-03-21T13:00:00Z",
  "total_classifications": 1234,
  "avg_score": 0.15,
  "avg_confidence": 0.82,
  "tier_counts": {
    "simple": 400,
    "moderate": 300,
    "complex": 350,
    "reasoning": 184
  },
  "signal_counts": {
    "short_message": 400,
    "code_block": 350,
    "attachment_gate": 50
  }
}

// GET /routing/decisions?limit=5
{
  "total": 1234,
  "limit": 5,
  "decisions": [
    {
      "id": 1233,
      "timestamp": "2026-03-21T12:59:59Z",
      "tier": "complex",
      "score": 0.25,
      "confidence": 0.78,
      "signals": [
        {"name": "code_block", "contribution": 0.4}
      ],
      "model": "glm-4-plus"
    }
  ]
}
```

### RouterV2 集成

RouterV2 自动集成 metrics 和 recorder：

```go
// 创建 router 时自动连接全局 metrics/recorder
router := routing.NewV2(cfg)

// 使用 SelectModelWithRecord 记录决策
model, tier, result := router.SelectModelWithRecord(msg, history, primaryModel, agentID)

// 或手动设置
router.SetMetrics(customCollector)
router.SetRecorder(customRecorder)
```

### 单元测试

| 测试文件 | 测试数量 | 覆盖内容 |
|---------|---------|---------|
| `metrics_test.go` | 6 | 基本记录、Reset、并发安全、NoOp、全局设置、直方图 |
| `recorder_test.go` | 9 | Ring buffer、按 tier 过滤、统计、隐私保护、禁用状态 |

**测试覆盖**：
- ✅ 基本记录和检索
- ✅ Ring buffer 行为
- ✅ 并发安全性
- ✅ 隐私保护验证（确保 RawValue 被剥离）
- ✅ 统计计算
- ✅ 重置功能

## 参考资料

- MoonHub 原有 2-tier 路由实现
