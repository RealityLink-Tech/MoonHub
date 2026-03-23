# SHIELD.md 反恶意软件 - 实现状态

**源码包**：`pkg/shield/` · **包内文档流**：[pkg/shield/docs/README.md](../../pkg/shield/docs/README.md)（[`SHIELD_MD.md`](../../pkg/shield/docs/SHIELD_MD.md)、[`EXAMPLES.md`](../../pkg/shield/docs/EXAMPLES.md)） · **仓库文档索引**：[docs/README.md](../README.md)

## 实现概述

MoonHub 的 SHIELD.md 反恶意软件系统已成功实现，为 MoonHub 添加了运行时威胁评估引擎。通过解析 YAML 格式的威胁定义，在工具执行前进行模式匹配，实现安全策略的实时执行。

## 已完成的工作

### 1. 核心包结构

```
pkg/shield/
├── types.go           # 核心类型定义 (ShieldAction, ThreatCategory, ShieldScope, ThreatEntry, ShieldEvent, ShieldDecision, Directive, MatchResult)
├── parser.go          # YAML 威胁解析器 (Parse, parseThreatBlock, ParseDirectives)
├── matcher.go         # 条件匹配引擎 (evaluateCondition, matchToolCall, matchToolArgsContaining, matchFilePathContains, etc.)
├── engine.go          # 决策引擎 (ShieldEngine, Evaluate, applyConfidenceThreshold)
├── approval.go        # 审批流程管理 (ApprovalManager, ApprovalRequest, ApprovalStatus)
├── default_threats.go # 内置默认威胁库 (8 个内置威胁)
├── parser_test.go     # 解析器单元测试
├── matcher_test.go    # 匹配器单元测试
├── engine_test.go     # 引擎单元测试
└── approval_test.go   # 审批流程单元测试
```

### 2. 核心功能

#### 2.1 威胁动作类型 (ShieldAction)
- `block`: 直接阻止操作
- `require_approval`: 需要用户审批（已完整实现）
- `log`: 仅记录日志

#### 2.2 威胁类别 (ThreatCategory)
- `tool`: 工具调用相关
- `skill`: 技能执行相关
- `network`: 网络出站相关
- `file`: 文件操作相关
- `prompt`: 提示词相关
- `memory`: 记忆相关
- `message`: 消息相关
- `other`: 其他

#### 2.3 事件范围 (ShieldScope)
- `tool.call`: 工具调用
- `skill.install`: 技能安装
- `skill.execute`: 技能执行
- `network.egress`: 网络出站
- `secrets.read`: 密钥读取
- `prompt`: 提示词

#### 2.4 置信度阈值机制
- 阈值: 0.85
- 置信度 >= 0.85: 执行原始动作
- 置信度 < 0.85: 降级为 `require_approval`
- Critical 严重级别: 覆盖置信度阈值，直接执行原始动作

#### 2.5 动作优先级
- `block` > `require_approval` > `log`
- 多个威胁匹配时，选择最高优先级的动作

#### 2.6 条件语法支持
```
BLOCK: tool.call <tool_name>                           # 工具名匹配
BLOCK: tool.call with arguments containing (DROP, DELETE)  # 参数内容匹配
APPROVE: skill name equals <value>                     # 技能名精确匹配
APPROVE: skill name contains <value>                   # 技能名包含匹配
LOG: outbound request to <domain>                      # 出站域名匹配
BLOCK: file path equals <path>                         # 文件路径精确匹配
BLOCK: file path contains <path>                       # 文件路径包含匹配
BLOCK: secrets read path equals <path>                 # 密钥路径匹配 (支持通配符)
LOG: incoming message contains <text>                  # 消息内容匹配
LOG: message contains <text>                           # 消息内容匹配 (别名)
```

### 3. 集成点

#### 3.1 pkg/agent/instance.go
- 在 `AgentInstance` 结构体中添加 `Shield *shield.ShieldEngine` 字段
- 在 `AgentInstance` 结构体中添加 `ApprovalManager *shield.ApprovalManager` 字段
- 在 `NewAgentInstance` 中初始化 Shield 引擎和审批管理器:
  - 优先加载 `workspace/SHIELD.md`
  - 不存在时使用默认威胁库

```go
type AgentInstance struct {
    // ... existing fields ...
    Shield          *shield.ShieldEngine
    ApprovalManager *shield.ApprovalManager
}

// 初始化逻辑
var shieldEngine *shield.ShieldEngine
shieldPath := filepath.Join(workspace, "SHIELD.md")
if _, err := os.Stat(shieldPath); err == nil {
    shieldEngine, _ = shield.NewEngineFromFile(shieldPath)
} else {
    shieldEngine = shield.NewEngineWithDefaults()
}

// 初始化审批管理器
approvalManager := shield.NewApprovalManager(5 * time.Minute)
```

#### 3.2 pkg/agent/loop.go
- 在工具执行前添加威胁评估 (约 line 1395)
- 完整实现 `require_approval` 动作的审批流程:
  - 创建审批请求
  - 向用户发送审批通知
  - 等待用户响应（支持超时）
  - 根据响应决定是否继续执行

```go
// Evaluate tool call against shield before execution
var toolResult *tools.ToolResult
if agent.Shield != nil && agent.Shield.IsActive() {
    decision := agent.Shield.Evaluate(shield.ShieldEvent{
        Scope:    shield.ScopeToolCall,
        ToolName: tc.Name,
        ToolArgs: tc.Arguments,
    })
    switch decision.Action {
    case shield.ActionBlock:
        toolResult = &tools.ToolResult{
            ForLLM:  fmt.Sprintf("Security policy blocked this tool call: %s", decision.Reason),
            IsError: true,
        }
        // ... logging and return
    case shield.ActionRequireApproval:
        // 创建审批请求
        req := agent.ApprovalManager.CreateRequest(event, decision)
        // 向用户发送审批通知
        approvalMsg := fmt.Sprintf("⚠️ **Approval Required**\n\nTool: `%s`\nReason: %s\n\nPlease respond with `approve %s` or `reject %s`",
            tc.Name, decision.Reason, req.ID, req.ID)
        al.bus.PublishOutbound(ctx, bus.OutboundMessage{...})
        // 等待用户响应
        approved, err := agent.ApprovalManager.WaitForApproval(ctx, req.ID)
        if err != nil || !approved {
            // 拒绝或超时
            return
        }
        // 用户批准，继续执行
    case shield.ActionLog:
        // Log and continue execution
    }
}
```

#### 3.3 pkg/commands/cmd_approve.go
- 新增 `/approve <approval_id>` 命令，用户可批准待处理的请求
- 新增 `/reject <approval_id>` 命令，用户可拒绝待处理的请求

#### 3.4 pkg/tools/skills_install.go
- 在 `InstallSkillTool` 中添加 Shield 评估
- 技能安装前检查是否被威胁库阻止

```go
// Shield evaluation for skill installation
if t.shield != nil && t.shield.IsActive() {
    decision := t.shield.Evaluate(shield.ShieldEvent{
        Scope:     shield.ScopeSkillInstall,
        SkillName: slug,
        ToolArgs:  args,
    })
    switch decision.Action {
    case shield.ActionBlock:
        return ErrorResult(fmt.Sprintf("Skill installation blocked by security policy: %s", decision.Reason))
    case shield.ActionRequireApproval:
        return &ToolResult{
            ForLLM:           fmt.Sprintf("Skill installation requires approval: %s", decision.Reason),
            RequiresApproval: true,
        }
    case shield.ActionLog:
        // Log and continue
    }
}
```

#### 3.5 pkg/tools/web.go
- 在 `WebFetchTool` 中添加 Shield 评估
- 网络请求前检查目标域名是否被威胁库阻止

```go
// Shield evaluation for network egress
if t.shield != nil && t.shield.IsActive() {
    decision := t.shield.Evaluate(ShieldEvent{
        Scope:  ScopeNetworkEgress,
        URL:    urlStr,
        Domain: hostname,
    })
    switch decision.Action {
    case ActionBlock:
        return ErrorResult(fmt.Sprintf("Network request blocked by security policy: %s", decision.Reason))
    case ActionRequireApproval:
        return &ToolResult{
            ForLLM:           fmt.Sprintf("Network request requires approval: %s", decision.Reason),
            RequiresApproval: true,
        }
    case ActionLog:
        // Log and continue
    }
}
```

### 4. 内置默认威胁库

| ID | Fingerprint | Category | Severity | Confidence | Action | Title |
|----|-------------|----------|----------|------------|--------|-------|
| THREAT-001 | sql-inject-01 | tool | high | 0.90 | block | SQL Injection via Tool Arguments |
| THREAT-002 | cmd-inject-01 | tool | critical | 0.95 | block | Command Injection |
| THREAT-003 | path-traversal-01 | file | high | 0.90 | block | Path Traversal Attack |
| THREAT-004 | sensitive-files-01 | file | high | 0.90 | block | Sensitive File Access |
| THREAT-005 | credential-access-01 | tool | high | 0.90 | block | Credential File Access |
| THREAT-006 | prompt-inject-01 | prompt | medium | 0.80 | log | Prompt Injection Detection |
| THREAT-007 | dangerous-exec-01 | tool | critical | 0.95 | block | Dangerous Command Execution |
| THREAT-008 | data-exfil-01 | network | high | 0.85 | log | Potential Data Exfiltration |

### 5. SHIELD.md 文件格式

```markdown
# SHIELD.md - Security Threat Definition File

```yaml
id: THREAT-001
fingerprint: sql-inject-01
category: tool
severity: high
confidence: 0.90
action: block
title: SQL Injection via Tool Arguments
description: Block SQL injection attempts in tool arguments
recommendation_agent: |
  BLOCK: tool.call with arguments containing (DROP, DELETE, UNION, --, ' OR '1'='1)
```

```yaml
id: THREAT-002
fingerprint: path-traversal-01
category: file
severity: high
confidence: 0.90
action: block
title: Path Traversal Attack
description: Block path traversal attempts
recommendation_agent: |
  BLOCK: file path contains ../
  BLOCK: file path contains /etc/passwd
  BLOCK: file path contains /etc/shadow
```
```

### 6. 审批流程系统 (新增)

#### 6.1 ApprovalManager
- 管理审批请求的生命周期
- 支持创建、批准、拒绝、超时处理
- 并发安全

#### 6.2 ApprovalRequest
- 唯一 ID 标识
- 关联的威胁 ID
- 工具名称和参数
- 创建时间和过期时间
- 当前状态 (pending/approved/rejected/expired)

#### 6.3 ApprovalStatus
- `pending`: 等待用户响应
- `approved`: 用户已批准
- `rejected`: 用户已拒绝
- `expired`: 请求已过期

#### 6.4 用户命令
- `/approve <approval_id>` - 批准请求
- `/reject <approval_id>` - 拒绝请求

## 使用示例

### 编程方式使用

```go
// 创建引擎（使用默认威胁）
engine := shield.NewEngineWithDefaults()

// 或从文件加载
engine, err := shield.NewEngineFromFile("SHIELD.md")

// 评估事件
decision := engine.Evaluate(shield.ShieldEvent{
    Scope:    shield.ScopeToolCall,
    ToolName: "db_query",
    ToolArgs: map[string]any{"query": "DROP TABLE users"},
})

if decision.Action == shield.ActionBlock {
    // 阻止操作
    return fmt.Errorf("blocked: %s", decision.Reason)
}

if decision.Action == shield.ActionRequireApproval {
    // 创建审批请求
    approvalMgr := shield.NewApprovalManager(5 * time.Minute)
    req := approvalMgr.CreateRequest(event, decision)

    // 等待用户批准
    approved, err := approvalMgr.WaitForApproval(ctx, req.ID)
    if !approved {
        return fmt.Errorf("rejected: %s", decision.Reason)
    }
}
```

### 运行时热更新

```go
// 重新加载 SHIELD.md
engine.Reload(newContent)

// 动态添加威胁
engine.AddThreat(shield.ThreatEntry{
    ID:                  "THREAT-CUSTOM-001",
    Category:            shield.CategoryTool,
    Severity:            shield.SeverityHigh,
    Confidence:          0.90,
    Action:              shield.ActionBlock,
    Title:               "Custom Threat",
    RecommendationAgent: "BLOCK: tool.call dangerous_tool",
})

// 移除威胁
engine.RemoveThreat("THREAT-001")

// 获取当前威胁列表
threats := engine.GetThreats()
```

## 单元测试

### 测试覆盖

- **parser_test.go**: 5 个测试函数，覆盖 YAML 解析、指令解析、有效性验证、过期检查
- **matcher_test.go**: 7 个测试函数，覆盖工具调用匹配、参数匹配、文件路径匹配、范围兼容性、模式提取
- **engine_test.go**: 10 个测试函数，覆盖引擎创建、评估逻辑、置信度阈值、严重级别覆盖、动作优先级、热更新
- **approval_test.go**: 9 个测试函数，覆盖审批管理器创建、请求创建、批准/拒绝、过期处理、并发访问

### 测试结果

```
=== 所有 33 个测试通过 ===
ok      github.com/sipeed/moonhub/pkg/shield
```

## 性能目标

- 威胁解析: 启动时一次性完成
- 运行时评估: < 1ms 每次工具调用
- 内存占用: < 1MB (典型威胁库)

## 已修复的问题

### 1. 条件匹配顺序问题
- **问题**: `tool.call with arguments containing` 模式无法匹配
- **原因**: `tool.call ` 前缀检查在 `tool.call with arguments containing ` 之前，导致后者永远不会被检测到
- **修复**: 调整条件检查顺序，将更具体的模式放在前面

### 2. 测试内容解析问题
- **问题**: `TestEvaluateBlock` 测试失败
- **原因**: Go 原始字符串拼接导致 YAML 解析问题
- **修复**: 使用内联字符串格式，显式使用 `\n` 换行符

## 文件清单

### 新建文件
- `pkg/shield/types.go`
- `pkg/shield/parser.go`
- `pkg/shield/matcher.go`
- `pkg/shield/engine.go`
- `pkg/shield/approval.go`
- `pkg/shield/default_threats.go`
- `pkg/shield/parser_test.go`
- `pkg/shield/matcher_test.go`
- `pkg/shield/engine_test.go`
- `pkg/shield/approval_test.go`
- `pkg/commands/cmd_approve.go`

### 修改文件
- `pkg/shield/types.go` - 添加 URL、Domain、ApprovalID 字段到 ShieldEvent，添加 ApprovalRequest 到 ShieldDecision
- `pkg/agent/instance.go` - 添加 Shield 和 ApprovalManager 字段，添加 time 导入
- `pkg/agent/loop.go` - 完整实现审批流程，更新 InstallSkillTool 调用
- `pkg/commands/runtime.go` - 添加 ApproveAction 和 RejectAction 回调
- `pkg/commands/builtin.go` - 注册 approve 和 reject 命令
- `pkg/tools/result.go` - 添加 RequiresApproval 字段
- `pkg/tools/skills_install.go` - 添加 Shield 评估，更新构造函数签名
- `pkg/tools/web.go` - 添加 Shield 评估，添加 ShieldEvaluator 接口

## 待完成工作 (TODO)

### 1. 威胁库更新机制 (低优先级)
- 支持远程威胁库同步
- 支持威胁库版本管理

## 参考资料

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
