# 插件架构 - 实现状态

**包内文档**：[pkg/framework/docs/README.md](../../pkg/framework/docs/README.md) · [pkg/plugins/docs/README.md](../../pkg/plugins/docs/README.md) · **仓库文档索引**：[docs/README.md](../README.md)

## 实现概述

MoonHub 正在实现一个完整的插件架构。目标是让 channels、providers 和 tools 都成为可扩展的插件，保持核心精简。

## 设计决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 加载策略 | 内置插件 + init() 注册 | 编译时加载，通过 init() 函数自动注册 |
| 热更新 | 不需要 | 启动时确定，无需运行时动态加载 |
| 二进制 | 单文件 | 保持单一二进制部署 |

## 已完成的工作

### 1. 核心包结构

```
pkg/framework/
├── types.go           # 核心接口和元数据定义
├── channel.go         # Channel 插件接口
├── provider.go        # Provider 插件接口
├── tool.go            # Tool 插件接口
├── registry.go        # 插件注册系统
├── manager.go         # 插件生命周期管理
└── builtin.go         # 内置插件导入触发
```

### 2. 核心接口定义

#### 2.1 Plugin 基础接口 (`pkg/framework/types.go`)

```go
type PluginType string

const (
    TypeChannel  PluginType = "channel"
    TypeProvider PluginType = "provider"
    TypeTool     PluginType = "tool"
)

type Metadata struct {
    ID          string     // "moonhub-channel-telegram"
    Name        string     // "Telegram"
    Type        PluginType
    Version     string     // "1.0.0"
    Description string
    Priority    int        // 加载顺序
}

type Plugin interface {
    Metadata() Metadata
    Init(ctx *RuntimeContext) error
    Validate(cfg *config.Config) error
}

type RuntimeContext struct {
    Config     *config.Config
    Bus        *bus.MessageBus
    Workspace  string
    MediaStore media.MediaStore
}
```

#### 2.2 Channel 插件接口 (`pkg/framework/channel.go`)

```go
type Channel interface {
    Name() string
    Start(ctx interface{}) error
    Stop(ctx interface{}) error
    Send(ctx interface{}, msg interface{}) error
    IsRunning() bool
}

type ChannelPlugin interface {
    Plugin
    ChannelPrefix() string
    CreateChannel(cfg *config.Config, bus *bus.MessageBus) (Channel, error)
    IsEnabled(cfg *config.Config) bool
}
```

#### 2.3 Provider 插件接口 (`pkg/framework/provider.go`)

```go
type ProviderPlugin interface {
    Plugin
    ProviderName() string
    CreateProvider(cfg *config.Config) (providers.LLMProvider, error)
    CreateProviderFromModelConfig(modelCfg *config.ModelConfig) (providers.LLMProvider, error)
    IsEnabled(cfg *config.Config) bool
    SupportsProtocol(protocol string) bool
    SupportsModel(modelName string) bool
}
```

#### 2.4 Tool 插件接口 (`pkg/framework/tool.go`)

```go
type ToolPlugin interface {
    Plugin
    CreateTools(ctx *RuntimeContext) []tools.Tool
    IsCore() bool
}
```

### 3. 插件注册系统 (`pkg/framework/registry.go`)

```go
type Registry struct {
    mu        sync.RWMutex
    plugins   map[string]Plugin
    channels  map[string]ChannelPlugin
    providers map[string]ProviderPlugin
    tools     map[string]ToolPlugin
}

func RegisterPlugin(p Plugin)
func GlobalRegistry() *Registry
```

### 4. 已创建的 Channel 插件 (16 个)

所有 Channel 插件位于 `pkg/plugins/channels/<name>/plugin.go`：

| 插件 | ID | 状态 |
|------|-----|------|
| telegram | moonhub-channel-telegram | 已创建 |
| discord | moonhub-channel-discord | 已创建 |
| slack | moonhub-channel-slack | 已创建 |
| matrix | moonhub-channel-matrix | 已创建 |
| feishu | moonhub-channel-feishu | 已创建 |
| qq | moonhub-channel-qq | 已创建 |
| dingtalk | moonhub-channel-dingtalk | 已创建 |
| line | moonhub-channel-line | 已创建 |
| onebot | moonhub-channel-onebot | 已创建 |
| wecom | moonhub-channel-wecom | 已创建 |
| wecom_app | moonhub-channel-wecom_app | 已创建 |
| wecom_aibot | moonhub-channel-wecom_aibot | 已创建 |
| pico | moonhub-channel-pico | 已创建 |
| irc | moonhub-channel-irc | 已创建 |
| maixcam | moonhub-channel-maixcam | 已创建 |
| whatsapp | moonhub-channel-whatsapp | 已创建 |
| whatsapp_native | moonhub-channel-whatsapp_native | 已创建 |

### 5. Channel 插件示例结构

```go
// pkg/plugins/channels/telegram/plugin.go
package telegram

import (
    "github.com/RealityLink-Tech/MoonHub/pkg/framework"
    "github.com/RealityLink-Tech/MoonHub/pkg/config"
)

func init() {
    plugin.RegisterPlugin(&TelegramPlugin{})
}

type TelegramPlugin struct {
    ctx *plugin.RuntimeContext
}

func (p *TelegramPlugin) Metadata() plugin.Metadata {
    return plugin.Metadata{
        ID:          "moonhub-channel-telegram",
        Name:        "Telegram",
        Type:        plugin.TypeChannel,
        Version:     "2.0.0",
        Description: "Telegram bot channel integration",
        Priority:    100,
    }
}

func (p *TelegramPlugin) ChannelPrefix() string {
    return "telegram"
}

func (p *TelegramPlugin) IsEnabled(cfg *config.Config) bool {
    return cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token != ""
}

func (p *TelegramPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (plugin.Channel, error) {
    return channelstelegram.NewTelegramChannel(cfg, bus)
}

func (p *TelegramPlugin) Init(ctx *plugin.RuntimeContext) error {
    p.ctx = ctx
    return nil
}

func (p *TelegramPlugin) Validate(cfg *config.Config) error {
    if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token == "" {
        return fmt.Errorf("telegram token required when enabled")
    }
    return nil
}
```

## 已解决的问题

### 循环导入 (已解决)

原循环: `plugin` -> `plugins/channels/telegram` -> `channels` -> `plugin`

**解决方案**: 将插件导入从 `pkg/framework/builtin.go` 移至 `cmd/moonhub/internal/gateway/helpers.go`。Gateway 在加载 channels 后导入插件包，确保依赖顺序正确，打破循环。

## 待完成的工作

### Phase 1: 基础设施 (已完成)
- [x] 核心插件类型定义
- [x] Channel 插件接口
- [x] Provider 插件接口
- [x] Tool 插件接口
- [x] 插件注册系统
- [x] 插件管理器
- [x] 内置插件索引

### Phase 2: Channel 插件迁移 (已完成)
- [x] 创建 16 个 Channel 插件包装器
- [x] 修复循环导入问题
- [x] 更新 channel manager 使用插件系统
- [x] 验证构建成功

### Phase 3: Provider 插件迁移 (已完成)
- [x] 创建 provider 插件目录结构
- [x] 包装每个 provider (openai_compat, anthropic, anthropic_messages, antigravity, claude_cli, codex_cli, github_copilot, openai_oauth)
- [x] 更新 provider factory 使用插件注册 (SetPluginProviderResolver)

### Phase 4: Tool 插件迁移 (已完成)
- [x] 创建 tool 插件目录结构
- [x] 迁移 web (web_search, web_fetch) 和 message 到插件包
- [x] 更新 tool 注册使用插件管理器 (InitializeToolsOnly, MergeFrom)

### Phase 5: 核心系统更新 (已完成)
- [x] 保留 built-in factory 作为 fallback
- [x] 更新 plugin.LoadBuiltin 注释
- [x] 构建和测试验证
- [x] 删除遗留的 factory 模式代码 (`pkg/channels/manager.go:initChannel`, `pkg/channels/registry.go.getFactory`) 以通过 CI 风格检查

- [x] 文档更新：状态文档更新

## 文件清单

### 新建文件
- `pkg/framework/types.go` - 核心接口
- `pkg/framework/channel.go` - Channel 接口
- `pkg/framework/provider.go` - Provider 接口
- `pkg/framework/tool.go` - Tool 接口
- `pkg/framework/registry.go` - 注册系统
- `pkg/framework/manager.go` - 管理器
- `pkg/framework/builtin.go` - 内置插件导入
- `pkg/plugins/channels/telegram/plugin.go`
- `pkg/plugins/channels/discord/plugin.go`
- `pkg/plugins/channels/slack/plugin.go`
- `pkg/plugins/channels/matrix/plugin.go`
- `pkg/plugins/channels/feishu/plugin.go`
- `pkg/plugins/channels/qq/plugin.go`
- `pkg/plugins/channels/dingtalk/plugin.go`
- `pkg/plugins/channels/line/plugin.go`
- `pkg/plugins/channels/onebot/plugin.go`
- `pkg/plugins/channels/wecom/plugin.go`
- `pkg/plugins/channels/wecom_app/plugin.go`
- `pkg/plugins/channels/wecom_aibot/plugin.go`
- `pkg/plugins/channels/irc/plugin.go`
- `pkg/plugins/channels/maixcam/plugin.go`
- `pkg/plugins/channels/whatsapp/plugin.go`
- `pkg/plugins/channels/whatsapp_native/plugin.go`

### 新建文件 (Phase 3-4)
- `pkg/plugins/providers/openai_compat/plugin.go`
- `pkg/plugins/providers/openai_oauth/plugin.go`
- `pkg/plugins/providers/anthropic/plugin.go`
- `pkg/plugins/providers/anthropic_messages/plugin.go`
- `pkg/plugins/providers/antigravity/plugin.go`
- `pkg/plugins/providers/claude_cli/plugin.go`
- `pkg/plugins/providers/codex_cli/plugin.go`
- `pkg/plugins/providers/github_copilot/plugin.go`
- `pkg/plugins/tools/web/plugin.go`
- `pkg/plugins/tools/message/plugin.go`

### 已修改文件
- `pkg/framework/builtin.go` - 移除插件导入，LoadBuiltin 改为 no-op
- `pkg/framework/provider.go` - 新增 CreateProviderFromModelConfig, SupportsProtocol
- `pkg/framework/manager.go` - 新增 InitializeToolsOnly，initProviders 支持 ErrSkipProvider
- `pkg/providers/factory_provider.go` - 新增 SetPluginProviderResolver，CreateProviderFromConfig 优先尝试插件
- `cmd/moonhub/internal/gateway/helpers.go` - 添加 channel/provider/tool 插件导入，设置 provider resolver，创建 plugin manager
- `pkg/channels/manager.go` - 使用插件系统初始化 channels
- `pkg/agent/loop.go` - NewAgentLoopWithPluginTools，registerSharedTools 支持插件工具合并
- `pkg/tools/registry.go` - 新增 MergeFrom 方法

## 包内文档（文档流）

与 `pkg/learning/docs/` 类似，实现目录旁提供分层说明，便于从代码侧阅读：

- [`pkg/framework/docs/README.md`](../../pkg/framework/docs/README.md) — 插件核心类型与 Registry / Manager  
- [`pkg/plugins/docs/README.md`](../../pkg/plugins/docs/README.md) — 内置插件目录与新增插件步骤  

## 验证步骤

1. **构建**: `make build` - 应该编译无错误
2. **测试**: `make test` - 所有现有测试应该通过
3. **功能测试**: 启动 gateway 并启用单个 channel
   - 验证 channel 通过插件系统初始化
   - 验证消息正确流转

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        MoonHub Core                         │
├─────────────────────────────────────────────────────────────┤
│  cmd/moonhub                                                │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────┐     ┌──────────────┐     ┌────────────────┐   │
│  │ Config  │────▶│    Plugin    │────▶│ Plugin Manager │   │
│  └─────────┘     │   Registry   │     └────────────────┘   │
│                  └──────────────┘             │             │
│                           │                   │             │
│                           ▼                   ▼             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Plugins                            │   │
│  ├─────────────────┬─────────────────┬─────────────────┤   │
│  │ Channel Plugins │ Provider Plugins│   Tool Plugins  │   │
│  ├─────────────────┼─────────────────┼─────────────────┤   │
│  │ - telegram      │ - openai        │ - web           │   │
│  │ - discord       │ - anthropic     │ - filesystem    │   │
│  │ - slack         │ - gemini        │ - cron          │   │
│  │ - matrix        │ - zhipu         │ - memory        │   │
│  │ - ... (16)      │ - ...           │ - ...           │   │
│  └─────────────────┴─────────────────┴─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 更新日志

### 2026-03-19
- 创建插件基础设施 (Phase 1 完成)
- 创建 16 个 Channel 插件包装器 (Phase 2 部分完成)
- 发现循环导入问题，正在解决

### 2026-03-19 (续)
- 修复循环导入：将插件导入移至 gateway/helpers.go
- 修复 16 个 channel 插件的导入、构造函数调用和配置传递
- 修复 plugin.Manager 使用 Get*Plugins 替代 List*Plugins
- 修复 provider 接口使用 providers.LLMProvider
- Phase 2 完成，构建通过

### 2026-03-19 (Phase 3-5)
- Phase 3: Provider 插件迁移完成。新增 CreateProviderFromModelConfig、SupportsProtocol 接口；创建 8 个 provider 插件；通过 SetPluginProviderResolver 注入，factory 优先尝试插件后 fallback 到 built-in
- Phase 4: Tool 插件迁移完成。创建 web、message 插件；Gateway 调用 InitializeToolsOnly；AgentLoop 通过 MergeFrom 合并插件工具
- Phase 5: 文档更新，构建验证通过

### 2026-03-20 (review)
- 文档与仓库对齐：插件实现目录为 `pkg/plugins/...`（非 `pkg/frameworks/...`）；修正 `helpers.go` 中误写的 `pkg/frameworks/providers|tools` 导入，否则无法编译
- `CreateProviderFromConfig`：仅在 resolver 返回 `ErrSkipProvider` 时回退 built-in；其它错误应向上返回（避免吞掉插件真实错误）
