# 记忆体系 - 实现状态

**源码包**：`pkg/adaptive_memory/` · **仓库文档索引**：[docs/README.md](../README.md)

## 实现概述

MoonHub 的自适应记忆系统已成功实现，并对中文进行了特别优化。

## 已完成的工作

### 1. 核心包结构

```
pkg/adaptive_memory/
├── types.go           # 数据结构和接口定义
├── schema.go          # SQLite Schema
├── store.go           # 数据库操作
├── scoring.go         # 评分算法
├── chinese.go         # 中文分词接口
├── gse_adapter.go     # GSE 分词器适配
├── engine.go          # 核心引擎
├── consolidation.go   # 记忆整合
├── migration.go       # 自动迁移
└── engine_test.go     # 单元测试
```

### 2. 核心功能

#### 2.1 情景记忆 (Episodic Memory)
- 支持多种事件类型：
  - 基础类型: correction, preference_learned, fact_stored, task_completed, delegation_result
  - 扩展类型: user_feedback, insight, reminder, error_learned, context_update
- 基于事件类型的默认重要性评分
- 时间戳和访问计数跟踪

#### 2.2 语义搜索 (FTS5)
- SQLite FTS5 全文搜索
- unicode61 分词器
- 双内容存储（原始 + 分词后）

#### 2.3 时间衰减 (Temporal Decay)
- Ebbinghaus 遗忘曲线实现
- 公式：`e^(-0.05 * days) * (1 + 0.02 * access_count)`
- 访问频率强化

#### 2.4 中文优化
- go-ego/gse 分词库集成
- 懒加载分词器（节省内存）
- 智能检测中文内容
- 双存储策略（原始 + 分词）

### 3. 集成点

#### 3.1 pkg/agent/memory.go
- 集成自适应记忆引擎
- 自动迁移 MEMORY.md
- 新增方法：
  - `GetAdaptiveMemory()`
  - `RecordMemoryEvent()`
  - `SearchMemory()`
  - `ConsolidateMemory()`
  - `GetMemoryContextWithQuery()`

#### 3.2 pkg/agent/context.go
- 查询相关的记忆检索
- 在构建上下文时自动搜索相关记忆

#### 3.3 pkg/config/config.go
- 新增 `AdaptiveMemoryConfig` 配置结构
- 支持环境变量配置
- 默认配置优化

### 4. 配置选项

```json
{
  "adaptive_memory": {
    "enabled": true,
    "enable_chinese": true,
    "fts_max_results": 50,
    "context_max_results": 10,
    "decay_older_than_days": 7,
    "decay_factor": 0.95,
    "prune_importance_max": 0.1,
    "prune_access_count_max": 0,
    "prune_older_than_days": 30,
    "merge_similarity_threshold": 0.8
  }
}
```

### 5. 依赖

- `github.com/go-ego/gse v0.80.2` - 中文分词
- `modernc.org/sqlite` - SQLite（已有）

## 使用示例

### 记录记忆事件

```go
// 通过 MemoryStore
id, err := memoryStore.RecordMemoryEvent(
    adaptive_memory.EventTypeFactStored,
    "用户喜欢简洁的代码风格",
    nil,
    nil,
)
```

### 搜索记忆

```go
results, err := memoryStore.SearchMemory("代码风格", 10)
for _, result := range results {
    fmt.Printf("%s (相关性: %.2f)\n", result.Content, result.RelevanceScore)
}
```

### 整合记忆

```go
result, err := memoryStore.ConsolidateMemory()
fmt.Printf("衰减: %d, 修剪: %d, 合并: %d\n",
    result.Decayed, result.Pruned, result.Merged)
```

## 已完成的工作

### 4. 扩展事件类型 (新增)
- `user_feedback`: 用户反馈 (importance: 0.85)
- `insight`: Agent 生成的洞察 (importance: 0.7)
- `reminder`: 计划提醒 (importance: 0.6)
- `error_learned`: 错误模式学习 (importance: 0.75)
- `context_update`: 上下文更新 (importance: 0.5)

### 5. 记忆工具集成 (新增)
- 创建 `pkg/tools/memory_tool.go`
- 实现以下操作:
  - `record`: 记录新的记忆事件
  - `search`: 搜索相关记忆
  - `list`: 列出最近记忆
  - `consolidate`: 执行记忆整合
- 在 `pkg/agent/loop.go` 中注册 memory tool
- 在 `pkg/agent/memory.go` 中添加 `GetAdaptiveMemory()` 方法
- 在 `pkg/agent/context.go` 中添加 `GetMemoryStore()` 方法

- 工具通过 `config.tools.memory.enabled` 配置，默认启用

## 使用示例

### 记录记忆事件

```go
// 通过 MemoryTool (Agent 调用)
{
  "action": "record",
  "event_type": "preference_learned",
  "content": "用户喜欢简洁的代码风格"
}
```

### 搜索记忆
```go
{
  "action": "search",
  "query": "代码风格",
  "limit": 10
}
```

### 列出最近记忆
```go
{
  "action": "list",
  "limit": 10
}
```

### 整合记忆
```go
{
  "action": "consolidate"
}
```

## 优化性工作结果

1. **中文分词质量验证** ✅ 通过
   - 新增 `chinese_segmenter_test.go` - 9 个测试用例全部通过
   - 测试了基础分词、智能检测、搜索模式

   - 分词质量验证: GSE 分词器能正确识别中文词汇边界

2. **性能测试（内存使用)**
   - 新增 `performance_test.go` - 6 个测试函数
   - **内存使用报告:**
     - 无中文模式: ~3MB (符合预期)
     - 中文模式: ~190MB (GSE 字典加载)
     - 1000 条记录: ~3MB
   - **延迟测试:**
     - 录入延迟: ~0.8ms/条
     - 搜索延迟: ~2.7ms
     - 整合延迟: ~85ms
3. **更新文档**

## 文件清单

### 新建文件
- `pkg/adaptive_memory/types.go`
- `pkg/adaptive_memory/schema.go`
- `pkg/adaptive_memory/store.go`
- `pkg/adaptive_memory/scoring.go`
- `pkg/adaptive_memory/chinese.go`
- `pkg/adaptive_memory/gse_adapter.go`
- `pkg/adaptive_memory/gse_adapter_simple.go` (GSE 失败时的简单分词回退)
- `pkg/adaptive_memory/engine.go`
- `pkg/adaptive_memory/consolidation.go`
- `pkg/adaptive_memory/migration.go`
- `pkg/adaptive_memory/engine_test.go`
- `pkg/tools/memory_tool.go`
- `pkg/tools/memory_tool_test.go`

### 修改文件
- `pkg/agent/memory.go`
- `pkg/agent/context.go`
- `pkg/agent/loop.go`
- `pkg/config/config.go`
- `go.mod`
