# Delegation System 子 Agent 编排系统实现文档

**包内文档（配置与阅读顺序）**：[pkg/delegation/docs/README.md](../../pkg/delegation/docs/README.md) · [CONFIG.md](../../pkg/delegation/docs/CONFIG.md) · **源码包**：`pkg/delegation/` · **Agent 集成**：`pkg/agent/delegation_integration.go` · **仓库文档索引**：[docs/README.md](../README.md)

## 概述

Delegation System 是一个为 MoonHub 设计的子 Agent 编排系统。它实现了：

- **非阻塞委托**：主 Agent 可以将任务委托给子 Agent 后继续工作
- **角色模板复用**：自动学习和复用成功的任务配置
- **自适应超时**：基于历史数据估算任务超时时间
- **黑板协作模式**：多个子 Agent 可以协作完成任务
- **后台任务**：长时间运行的任务在后台执行

## 实现状态

### ✅ 已完成

| 组件 | 状态 | 说明 |
|------|------|------|
| 核心类型 | ✅ 完成 | types.go 定义所有核心类型 |
| SQLite 存储 | ✅ 完成 | store.go 实现 5 张表的持久化 |
| Intercom 事件系统 | ✅ 完成 | intercom.go 实现发布订阅 |
| 超时估算器 | ✅ 完成 | timeout_estimator.go 自适应超时 |
| 模板管理器 | ✅ 完成 | templates.go 角色模板管理 |
| 生命周期管理 | ✅ 完成 | lifecycle.go 子 Agent 生命周期 |
| 后台执行器 | ✅ 完成 | background.go 后台任务执行 |
| 会话队列 | ✅ 完成 | queue.go 串行/并行任务调度 |
| 工具工厂 | ✅ 完成 | tools.go 主系统集成 |
| 8 个工具实现 | ✅ 完成 | delegate_task, delegate_tasks 等 |
| 配置集成 | ✅ 完成 | config.go, defaults.go |
| Agent 集成 | ✅ 完成 | delegation_integration.go |

## 包结构

```
pkg/delegation/
├── types.go              # 核心类型定义
├── store.go              # SQLite 存储（5 张表）
├── intercom.go           # Pub/Sub 事件系统
├── timeout_estimator.go  # 自适应超时估算
├── templates.go          # 角色模板管理
├── lifecycle.go          # 子 Agent 生命周期
├── background.go         # 后台任务执行器
├── queue.go              # 会话队列
├── tools.go              # 工具工厂和主系统
├── tools_delegate.go     # delegate_task, delegate_tasks 工具
├── tools_background.go   # delegate_background, delegate_to_existing 工具
├── tools_manage.go       # list, manage, confirm 工具
└── tools_helper.go       # 共享辅助函数

pkg/agent/
└── delegation_integration.go  # Agent 集成代码
```

## 架构

### 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                    DelegationSystem                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐    │
│  │  Intercom   │  │  Blackboard  │  │  TimeoutEstimator │   │
│  │ (Pub/Sub)   │  │  (协作)       │  │  (自适应超时)      │    │
│  └─────────────┘  └──────────────┘  └──────────────────┘    │
│                                                              │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐    │
│  │  Lifecycle  │  │  Templates   │  │   Background     │    │
│  │  Manager    │  │  Manager     │  │   Runner         │    │
│  └─────────────┘  └──────────────┘  └──────────────────┘    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  SessionQueue                         │   │
│  │            (每个 Agent 串行执行)                        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   SQLite Store  │
                    │  (delegation.db)│
                    └─────────────────┘
```

### Intercom Topics

Intercom 事件系统支持以下 Topics：

| Topic | 说明 | 触发时机 |
|-------|------|---------|
| `task:queued` | 任务入队 | 任务提交到后台队列 |
| `task:completed` | 任务完成 | 后台任务执行成功 |
| `task:failed` | 任务失败 | 后台任务执行失败 |
| `agent:created` | Agent 创建 | 新子 Agent 创建 |
| `agent:dismissed` | Agent 解雇 | 子 Agent 被解雇 |
| `agent:suspended` | Agent 挂起 | 子 Agent 被挂起 |
| `agent:revived` | Agent 恢复 | 子 Agent 恢复活动 |
| `memory:updated` | 内存更新 | 内存条目更新 |
| `memory:consolidated` | 内存整合 | 内存整合完成 |
| `blackboard:proposal` | 黑板提案 | 子 Agent 提交提案 |
| `blackboard:resolved` | 黑板决议 | 提案被解决 |
| `nudge:scheduled` | 提醒调度 | 提醒被调度 |
| `nudge:delivered` | 提醒送达 | 提醒已送达 |
| `nudge:suppressed` | 提醒抑制 | 提醒被抑制 |

### 数据库 Schema

```sql
-- 1. sub_agents: 子 Agent 持久化
CREATE TABLE sub_agents (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    label TEXT NOT NULL,
    role_prompt TEXT NOT NULL,
    task_keywords TEXT,
    tools TEXT,
    status TEXT DEFAULT 'active',
    created_at INTEGER,
    last_active_at INTEGER,
    completed_tasks INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0,
    session_key TEXT
);

-- 2. role_templates: 角色模板
CREATE TABLE role_templates (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    role_prompt TEXT NOT NULL,
    task_keywords TEXT,
    tools TEXT,
    category TEXT,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    avg_duration_ms INTEGER DEFAULT 0,
    created_at INTEGER,
    last_used_at INTEGER
);

-- 3. background_tasks: 后台任务
CREATE TABLE background_tasks (
    id TEXT PRIMARY KEY,
    sub_agent_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    task TEXT NOT NULL,
    category TEXT,
    status TEXT DEFAULT 'pending',
    result TEXT,
    created_at INTEGER,
    completed_at INTEGER,
    delivered INTEGER DEFAULT 0,
    origin_channel TEXT,
    origin_chat_id TEXT
);

-- 4. task_metrics: 超时估算指标
CREATE TABLE task_metrics (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    task_category TEXT NOT NULL,
    task_hint TEXT,
    duration_ms INTEGER,
    success INTEGER,
    created_at INTEGER
);

-- 5. sub_agent_messages: 子 Agent 消息历史
CREATE TABLE sub_agent_messages (
    id TEXT PRIMARY KEY,
    sub_agent_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at INTEGER
);
```

## 8 个委托工具

| 工具 | 功能 |
|------|------|
| `delegate_task` | 委托单个任务，自动复用/创建子 Agent |
| `delegate_tasks` | 批量委托（最多 10 个任务） |
| `delegate_background` | 后台委托，不阻塞主 Agent |
| `delegate_to_existing` | 向现有子 Agent 发送任务 |
| `list_sub_agents` | 列出所有活跃的子 Agent |
| `manage_sub_agent` | dismiss/revive/kill 子 Agent |
| `manage_template` | 模板 CRUD 操作 |
| `confirm_task` | 确认并获取后台任务结果 |

### 默认安全工具

子 Agent 默认只能使用以下安全工具：

```go
[]string{"read_file", "list_dir", "web_search", "web_fetch", "memory_recall"}
```

## 配置

### pkg/config/config.go

```go
type DelegationConfig struct {
    Enabled              bool     `json:"enabled"`
    DefaultSubAgentTools []string `json:"default_sub_agent_tools"`
    MaxActivePerUser     int      `json:"max_active_per_user"`
    MaxConcurrentTasks   int      `json:"max_concurrent_tasks"`
    RetentionDays        int      `json:"retention_days"`
    ReuseThreshold       float64  `json:"reuse_threshold"`
}
```

### 默认值

```go
DelegationConfig{
    Enabled:              false,  // 默认禁用
    DefaultSubAgentTools: []string{"read_file", "list_dir", "web_search", "web_fetch", "memory_recall"},
    MaxActivePerUser:     10,
    MaxConcurrentTasks:   3,
    RetentionDays:        14,
    ReuseThreshold:       0.6,
}
```

### config.json 示例

```json
{
  "delegation": {
    "enabled": true,
    "default_sub_agent_tools": ["read_file", "list_dir", "web_search", "web_fetch"],
    "max_active_per_user": 10,
    "max_concurrent_tasks": 3,
    "retention_days": 14,
    "reuse_threshold": 0.6
  }
}
```

## 集成点

### 1. AgentInstance 初始化

在 `pkg/agent/instance.go` 中，当 `cfg.Delegation.Enabled` 时调用 `NewDelegationIntegration`（将 `config.DelegationConfig` 转为包内配置）、`RegisterTools` 注入 8 个工具，并设置 `AgentInstance.Delegation`。

### 2. Agent Loop 注入后台任务结果与工具上下文

在 `pkg/agent/loop.go` 的 `runAgentLoop` 中，在 `BuildMessages` 之前对 `opts.UserMessage` 调用 `InjectBackgroundTaskResults`（分区 ID 与 `delegationUserIDForOpts(opts)` 一致，通常为 `deleg:` + 小写 `SessionKey`）。工具执行经 `ExecuteWithContext(..., delegationUserIDForOpts(opts), ...)` 注入 `tools.DelegationUserID(ctx)`，与存储/查询子 Agent、后台任务、模板一致。

### 3. 工具注册

委托工具在 Agent 创建时注册到该 Agent 的 `ToolRegistry`，无需在 `registerSharedTools` 中重复注册。

## 自适应超时估算

超时估算器根据历史数据自动计算任务超时时间：

1. **任务分类**：research, code, analysis, writing, general
2. **加权平均**：最近的任务权重更高
3. **置信度缓冲**：低置信度 = 更多缓冲时间
4. **限制**：最小 30 秒，最大 30 分钟

```go
// 示例：估算超时
timeout := timeoutEstimator.Estimate(ctx, userID, task, category)
// timeout.Duration = 5 * time.Minute
// timeout.Confidence = 0.8
```

## 角色模板复用

模板管理器通过关键词匹配自动复用角色模板：

1. **关键词提取**：从任务描述中提取关键词
2. **相似度计算**：Jaccard 相似度
3. **阈值匹配**：默认 60% 重叠即可复用
4. **自动创建**：成功任务后自动创建模板

```go
// 示例：查找或创建模板
template, reused, err := templates.FindOrCreate(ctx, userID, task, customPrompt, customTools)
// reused = true 表示复用了现有模板
```

## 使用示例

### 1. 委托单个任务

```json
{
  "tool": "delegate_task",
  "arguments": {
    "task": "Research the latest developments in Go 1.22",
    "label": "go-researcher",
    "category": "research"
  }
}
```

### 2. 批量委托

```json
{
  "tool": "delegate_tasks",
  "arguments": {
    "tasks": [
      {"task": "Analyze the error logs", "category": "analysis"},
      {"task": "Write unit tests for auth module", "category": "code"},
      {"task": "Summarize the meeting notes", "category": "writing"}
    ],
    "parallel": true
  }
}
```

### 3. 后台委托

```json
{
  "tool": "delegate_background",
  "arguments": {
    "task": "Run comprehensive code analysis and generate report",
    "label": "code-analyzer",
    "category": "analysis"
  }
}
```

### 4. 查看后台任务结果

```json
{
  "tool": "confirm_task",
  "arguments": {
    "mark_delivered": true
  }
}
```

### 5. 管理子 Agent

```json
{
  "tool": "manage_sub_agent",
  "arguments": {
    "sub_agent_id": "agent-123",
    "action": "dismiss"
  }
}
```

## 测试

```bash
# 运行单元测试
go test ./pkg/delegation/... -v

# 运行覆盖率测试
go test ./pkg/delegation/... -cover

# 运行基准测试
go test ./pkg/delegation/... -bench=. -benchmem
```

## 参考

技术特点：
1. 纯 Go 实现
2. SQLite 持久化
3. 自适应超时估算
4. 角色模板自动学习和复用
5. 黑板协作模式

## 实现文件清单

### 核心实现文件

| 文件路径 | 说明 |
|---------|------|
| `pkg/delegation/types.go` | 核心类型定义 |
| `pkg/delegation/store.go` | SQLite 存储实现 |
| `pkg/delegation/intercom.go` | Pub/Sub 事件系统 |
| `pkg/delegation/timeout_estimator.go` | 自适应超时估算 |
| `pkg/delegation/templates.go` | 角色模板管理 |
| `pkg/delegation/lifecycle.go` | 子 Agent 生命周期 |
| `pkg/delegation/background.go` | 后台任务执行器 |
| `pkg/delegation/queue.go` | 会话队列 |
| `pkg/delegation/tools.go` | 工具工厂和主系统 |
| `pkg/delegation/tools_delegate.go` | delegate_task, delegate_tasks 工具 |
| `pkg/delegation/tools_background.go` | delegate_background, delegate_to_existing 工具 |
| `pkg/delegation/tools_manage.go` | list, manage, confirm 工具 |
| `pkg/delegation/tools_helper.go` | 共享辅助函数 |

### 集成修改

| 文件路径 | 修改内容 |
|---------|---------|
| `pkg/config/config.go` | 添加 DelegationConfig 类型 |
| `pkg/config/defaults.go` | 添加 Delegation 默认配置 |
| `pkg/agent/delegation_integration.go` | Agent 集成代码 |

## 更新日志

- **2026-03-21**: Agent 运行时接线
  - `AgentInstance` 在 `delegation.enabled` 时初始化并注册 8 个委托工具；`AgentLoop` 在每轮 `runAgentLoop` 注入未投递的后台任务结果，并通过 `ExecuteWithContext` 传入与 session 一致的 delegation 分区 ID
  - `Blackboard.SetIntercom` 在 `NewDelegationSystem` 中自动连接；`Intercom.On` / `OnAny` 退订改为按订阅 ID 移除，避免多订阅者错位
  - 工具上下文新增 `DelegationUserID`；同步委托经 `SessionQueue` 按分区串行化

- **2026-03-21**: Intercom 事件系统完善
  - 添加完整的 Intercom Topics
  - 新增 `task:queued` 任务入队事件
  - 新增 `memory:updated/consolidated` 内存事件
  - 新增 `blackboard:resolved` 黑板决议事件
  - 新增 `nudge:*` 提醒事件系列
  - 完善 Blackboard 协作模式（支持 Resolve 方法）
  - 添加事件订阅辅助函数

- **2026-03-21**: 初始实现完成
  - 完成所有核心组件
  - 实现 8 个委托工具
  - 集成到配置系统
