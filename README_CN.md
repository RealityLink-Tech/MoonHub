# MoonHub

> [!NOTE]
> **致谢**
>
> 本项目灵感来源于 [TinyClaw](https://github.com/wgtechlabs/tinyclaw) 的功能集成和 [PicoClaw](https://github.com/sipeed/picoclaw) 的轻量化设计，在此基础上继续发展独属于本项目的方向。

**文档索引**（插件、学习、压缩器、SHIELD、记忆、委托、配网等）：[`docs/README.md`](docs/README.md)。

**[English](README.md)**

## 简介

> **面向日常用户的开箱即用 Agent。**

MoonHub 是一款面向边缘计算的**本地优先 AI 助手**——简单、快速、安全。

我们相信，AI 不应该是少数技术专家的专属工具。MoonHub 专为**日常用户**打造，无需任何技术背景，插电即用。目前适配 Linux 平台，完美运行于树莓派、工业网关等各类嵌入式设备。

### 设计理念

| 原则 | 描述 |
|------|------|
| **Simple** | 零学习曲线。开箱即用，像使用家用电器一样简单。 |
| **Fast** | 极致轻量。<10MB 内存，1 秒冷启动，毫秒级响应。 |
| **Secure** | 本地优先。数据不出设备，隐私完全由你掌控。 |

### 核心特性

**🤖 Agent 协作引擎**

你是指挥官，Agent 团队为你服务。每个 Agent 各司其职，它们可以彼此通信、主动协作，在关键决策点向你请示。这不是简单的问答机器人，而是一个真正理解上下文、能自主推进任务的智能团队。

**🎨 动态 UI 生成**

告别"只能输出文字"的传统 Agent。MoonHub 能够根据你的需求，实时生成可视化交互界面——财务仪表盘、任务管理器、数据看板，一切随需而变。你描述想法，Agent 为你构建。

**📱 专属应用**

用户通过专属应用与设备交互。目前以 **PWA** 形式提供，支持快速安装、离线使用；后续将推出原生 **APP**，覆盖更多平台与使用场景。

### 无限可能

MoonHub 扎根于边缘计算场景，在保持极致轻量（<10MB 内存）的同时，兼顾流畅体验与完整功能。虽然项目有明确的核心演进方向，但架构设计上充分支持各类边缘场景的二次开发——无论是智慧农业、工业物联网，还是智能零售、家庭自动化，你都可以基于 MoonHub 快速构建专属的智能化解决方案。

| 场景 | 描述 |
|------|------|
| **智能灌溉** | 接入土壤湿度、气象传感器，Agent 根据实时数据动态调整灌溉策略，实现精准节水农业。 |
| **工业监控** | 部署于生产车间，实时采集设备状态，预测性维护告警，生成可视化运维看板。 |
| **智慧门店** | 连接客流统计、库存传感器，自动生成补货建议、销售分析报告，辅助经营决策。 |
| **能源管理** | 对接智能电表、光伏逆变器，实时优化用电策略，生成能耗报告与节能建议。 |
| **智能 CRM** | 集成客户数据、沟通记录，AI 分析客户画像，自动生成跟进提醒与销售机会洞察。 |
| **智能运维** | 接入服务器、应用监控数据，AI 识别异常模式，自动告警并生成故障诊断报告。 |
| **智能安防** | 对接摄像头、门窗传感器，AI 识别异常行为，实时推送告警并生成安全日志。 |
| **智慧养殖** | 接入水质、投喂设备，实时监测养殖环境，自动调节投喂量并生成生长分析报告。 |
| **智慧教室** | 接入考勤设备、互动大屏，自动记录出勤，辅助教师生成个性化学习报告与教学建议。 |
| **智能电商** | 对接订单、库存、物流系统，AI 分析销售趋势，自动生成补货建议与营销策略。 |

你的想象力，就是 MoonHub 的边界。

### 快速开始

1. **插电启动** — 设备开机后自动创建 WiFi 热点（`MoonHub-XXXX`）
2. **手机配网** — 连接热点，访问配网页面，配置 WiFi 并设置授权码
3. **安装 PWA** — 配网完成后引导安装 PWA 应用
4. **开始使用** — PWA 自动扫描本地设备，输入授权码即可开始对话、管理和配置

## 功能特性

### 已实现

- **自适应记忆** — 3 层记忆系统（情景记忆、语义 FTS5、时间衰减），随时间学习该记住什么、该忘记什么。
- **自我进化** — 行为模式检测系统，从用户反馈中学习，追踪工具使用偏好，持续进化模式。
- **插件架构** — 渠道、提供商、工具皆为插件。核心保持精简，其他皆可扩展。
- **上下文压缩器** — 4 层上下文压缩流水线（规则、去重、LLM 摘要、L0/L1/L2 分层），集成于 Agent 循环；参见 [`docs/implementation/compactor-status.md`](docs/implementation/compactor-status.md) 和 [`pkg/compactor/docs/`](pkg/compactor/docs/README.md)。
- **SHIELD.md 反恶意软件** — 运行时威胁评估引擎，支持 YAML 威胁解析、模式匹配、审批流程，内置 8 种威胁；参见 [`docs/implementation/shield-status.md`](docs/implementation/shield-status.md) 和 [`pkg/shield/docs/`](pkg/shield/docs/README.md)。
- **委托系统** — 子 Agent 编排（非阻塞与后台任务、模板复用、自适应超时、SQLite 持久化、Intercom 发布订阅）。通过 `config.json` 中的 `delegation.enabled` 启用（默认关闭）；参见 [`pkg/delegation/docs/README.md`](pkg/delegation/docs/README.md)、[`pkg/delegation/docs/CONFIG.md`](pkg/delegation/docs/CONFIG.md) 和 [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md)。
- **Agent 间通信（Intercom）** — 进程内发布订阅，用于委托时信号传递：按主题订阅（`On`）、通配订阅（`OnAny`）、有界主题保留（`Recent`/`RecentAll`）；参见 [`pkg/delegation/intercom.go`](pkg/delegation/intercom.go) 和 [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md) 中的 Intercom 章节。
- **智能路由 V2** — 4 层模型路由系统（简单/中等/复杂/推理），基于规则评分、特征提取、隐私安全指标。将简单查询路由到廉价模型，复杂查询路由到强大模型；参见 [`pkg/routing/docs/README.md`](pkg/routing/docs/README.md) 和 [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md)。
- **设备配网** — 零配置 WiFi 配网（热点、扫描/连接、诊断、自动与手动恢复、恢复出厂、授权码、SSE）。在 Web 启动器上通过 `MOONHUB_PROVISIONING_ENABLED=1` 启用；含 React 配网向导与可选 PWA 离线缓存。参见 [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md)、[`pkg/provisioning/docs/CONFIG.md`](pkg/provisioning/docs/CONFIG.md)、[`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md)。

### 计划中

- **PWA 前端** — 面向最终用户的完整 PWA，用于设备发现、配对与日常使用（超出配网向导范围）
- **动态 UI 生成** — 基于用户需求实时生成可视化组件（仪表盘、任务管理器、数据可视化）
- **原生 APP** — iOS 和 Android 原生移动应用

<details>
<summary><strong>已完成（点击展开）</strong></summary>

- ~~**自我进化** — 行为模式检测，让 Agent 在每次交互中变得更好。它与你共同成长。~~ → **已实现**
- ~~**插件架构** — 渠道、提供商、工具皆为插件。核心保持精简，其他皆可扩展。~~ → **已实现**
- ~~**上下文压缩器** — 4 层上下文压缩流水线，包含基于规则的预压缩、去重、LLM 摘要和分层摘要。~~ → **已实现**
- ~~**SHIELD.md 反恶意软件** — 运行时 SHIELD.md 执行引擎，支持威胁解析、模式匹配和内置反恶意软件保护。~~ → **已实现**
- ~~**委托系统** — 自主子 Agent 编排，具备自我改进的角色模板、黑板协作和自适应超时。~~ → **已实现**（可选启用；参见 [`pkg/delegation/docs/`](pkg/delegation/docs/README.md) 和 [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md)）
- ~~**智能路由** — 4 层查询分类器，将简单查询路由到廉价模型，复杂查询路由到强大模型，降低 LLM 成本。~~ → **已实现**（参见 [`pkg/routing/docs/`](pkg/routing/docs/README.md) 和 [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md)）
- ~~**Agent 间通信** — 轻量级发布订阅事件总线，用于 Agent 间实时通信，支持通配订阅和有界历史。~~ → **已实现**（委托系统的 **Intercom**，位于 [`pkg/delegation/intercom.go`](pkg/delegation/intercom.go)；随委托功能启用）
- ~~**设备配网** — 零配置 WiFi 配网、恢复、出厂重置、配网 UI。~~ → **已实现**（启动器可选启用；参见 [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md) 和 [`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md)）

</details>

## 更新日志

<details>
<summary><strong>2026-03-22 — 设备配网文档流</strong></summary>

#### 摘要

仓库文档现已与路由、委托、压缩器保持一致的**包级文档 → 实现状态**模式：英文 `pkg/provisioning/docs/`、中文深度解析 `docs/implementation/provisioning-status.md`，并在文档索引、Web 指南和 `CLAUDE.md` 中建立交叉链接。

#### 文档

- [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md) — 范围、源码地图、集成表（Web API、启动器、前端）
- [`pkg/provisioning/docs/CONFIG.md`](pkg/provisioning/docs/CONFIG.md) — 环境变量、`provisioning.json`、持久化键、HTTP/SSE 和浏览器令牌说明
- [`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md) — 文首添加与上述文档一致的阅读顺序；保留原有 API 和 UI 参考
- [`docs/README.md`](docs/README.md) — 子系统表格、首次阅读第 6 步、`docs/implementation/` 与 `pkg/` 索引条目
- [`web/README.md`](web/README.md) — 可选配网小节（路径 + 文档流）
- [`CLAUDE.md`](CLAUDE.md) — `provisioning/` 包说明、启动器环境变量与文档链接
- [`README.md`](README.md) / [`README_CN.md`](README_CN.md) — 功能列表中配网已归类为已实现并附文档链接

</details>

<details>
<summary><strong>2026-03-21 — 智能路由 V2（4 层模型路由）</strong></summary>

#### 摘要

4 层模型路由系统，基于规则分类，取代原有的 2 层（轻量/重量）系统。根据消息复杂度自动选择合适的 LLM。

#### 新功能

**智能路由 V2**（`pkg/routing/`）
- **4 层分类** — simple、moderate、complex、reasoning 四个层级，边界可配置
- **基于规则的评分** — 使用结构特征进行亚微秒级分类（无 API 调用）
- **特征提取** — Token 估算、代码块、工具调用、对话深度、附件
- **附件硬门控** — 多模态输入自动路由到 reasoning 层
- **置信度评分** — 基于 sigmoid 的每次分类置信度计算
- **信号追踪** — 每个决策包含可解释的调试信号
- **隐私安全指标** — 聚合统计，不存储消息内容
- **决策记录器** — 环形缓冲区存储近期决策，支持层级过滤
- **HTTP 端点** — `/metrics`、`/routing/decisions`、`/routing/stats`
- **向后兼容** — 当配置了 `light_model` 时回退到 2 层模式

#### 文档

- [`pkg/routing/docs/README.md`](pkg/routing/docs/README.md) — 概述、架构、快速开始
- [`pkg/routing/docs/CONFIG.md`](pkg/routing/docs/CONFIG.md) — 配置选项、层级映射、自定义边界
- [`pkg/routing/docs/FEATURES.md`](pkg/routing/docs/FEATURES.md) — 特征提取、评分权重、示例
- [`pkg/routing/docs/METRICS.md`](pkg/routing/docs/METRICS.md) — 指标收集、决策记录器、HTTP 端点
- [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md) — 完整实现状态
- [`docs/README.md`](docs/README.md) — 仓库文档索引（已更新）

#### 技术细节

- 评分范围：[-1.0, 1.0]，简单消息为负分
- 默认边界：simple [-1, -0.05)、moderate [-0.05, 0.15)、complex [0.15, 0.35)、reasoning [0.35, 1.0]
- 权重：短消息 (-0.10)、代码块 (+0.40)、长消息 (+0.35)、附件 (1.0 硬门控)
- 42 个单元测试，全部通过

#### 变更文件

- `pkg/routing/` — 核心实现（tier、classifier、router、features、metrics、recorder）
- `pkg/routing/docs/` — 新文档目录（README、CONFIG、FEATURES、METRICS）
- `pkg/config/config.go` — RoutingConfig 含 TierMapping 和 TierBoundariesConfig
- `pkg/agent/instance.go` — RouterV2、TierCandidates 字段和初始化
- `pkg/agent/loop.go` — 更新 selectCandidates 以支持 4 层路由
- `pkg/health/server.go` — 指标和决策的 HTTP 端点
- `docs/README.md` — 更新智能路由章节，添加完整文档链接

</details>

<details>
<summary><strong>2026-03-21 — 委托系统（子 Agent 编排）</strong></summary>

#### 摘要

子 Agent 委托系统支持自主工作流：八个工具、SQLite 存储、会话队列、Agent 循环中的后台任务注入，以及工具执行上下文中的 `DelegationUserID`。

#### 文档

- [`pkg/delegation/docs/README.md`](pkg/delegation/docs/README.md)、[`pkg/delegation/docs/CONFIG.md`](pkg/delegation/docs/CONFIG.md) — 包级文档（流程 + 配置）
- [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md) — 深度实现参考
- [`docs/README.md`](docs/README.md) — 仓库索引（委托子系统表格）

#### 代码（概要）

- `pkg/delegation/` — 核心实现
- `pkg/agent/delegation_integration.go`、`pkg/agent/instance.go`、`pkg/agent/loop.go` — 运行时集成
- `pkg/config/config.go`、`pkg/config/defaults.go` — `DelegationConfig`

</details>

<details>
<summary><strong>2026-03-21 — SHIELD.md 反恶意软件实现</strong></summary>

#### 新功能

**SHIELD.md 反恶意软件系统**（`pkg/shield/`）

运行时威胁评估引擎：

- **威胁解析器** — YAML 格式的 SHIELD.md 解析器，支持威胁定义、指令和元数据
- **模式匹配器** — 条件语法支持工具调用、文件路径、网络出口、技能操作
- **执行动作** — 三种动作类型：`block`、`require_approval`、`log`，基于优先级解决
- **审批流程** — `/approve` 和 `/reject` 命令用于用户确认操作，5 分钟超时
- **工具集成** — Shield 评估已集成到 `web_fetch` 和 `install_skill` 工具
- **默认威胁** — 8 种内置威胁，覆盖 SQL 注入、命令注入、路径遍历、凭证访问等

#### 技术细节

- 置信度阈值 (0.85)，关键威胁支持严重性覆盖
- 动作优先级：`block` > `require_approval` > `log`
- 基于上下文的审批绕过，防止重复评估
- 完善的单元测试（31 个测试，100% 通过率）

#### 变更文件

- `pkg/shield/` — 新包（11 个核心文件 + 5 个测试文件）
- `pkg/agent/instance.go` — Shield 和 ApprovalManager 初始化
- `pkg/agent/loop.go` — 工具执行流程中的 Shield 评估
- `pkg/commands/cmd_approve.go` — 审批/拒绝命令处理
- `pkg/tools/web.go` — 网络出口的 Shield 集成
- `pkg/tools/skills_install.go` — 技能安装的 Shield 集成
- `docs/implementation/shield-status.md` — 实现状态

</details>

<details>
<summary><strong>2026-03-20 — 上下文压缩器与文档流</strong></summary>

#### 功能

- **上下文压缩器**（`pkg/compactor/`）— 四层流水线（基于规则的预压缩、去重、LLM 摘要、L0/L1/L2 分层），集成于 Agent；参见 `compactor` 配置和 [`pkg/compactor/docs/CONFIG.md`](pkg/compactor/docs/CONFIG.md)。

#### 文档

- 添加仓库文档入口 [`docs/README.md`](docs/README.md)，区分"包级文档"与 `docs/implementation/*-status.md`，与 `pkg/learning/docs` 保持一致。
- 添加 [`pkg/compactor/docs/`](pkg/compactor/docs/README.md)（README + CONFIG）。
- 修复 `plugin-architecture-status.md` 链接指向实际文件 [`docs/implementation/plugin-status.md`](docs/implementation/plugin-status.md)。

</details>

<details>
<summary><strong>2025-03-20 — 插件架构实现</strong></summary>

#### 新功能

**插件架构系统**（`pkg/framework/`、`pkg/plugins/`）

全面的插件系统，使渠道、提供商和工具皆为可扩展插件：

- **核心框架** — 插件类型、接口（Channel/Provider/Tool）、注册系统、生命周期管理
- **渠道插件** — 16 个渠道插件迁移（Telegram、Discord、Slack、Matrix、飞书、QQ、钉钉、LINE、OneBot、企业微信、企业微信应用、企业微信 AI 机器人、Pico、IRC、MaixCam、WhatsApp）
- **提供商插件** — 8 个提供商插件迁移（OpenAI Compat、OpenAI OAuth、Anthropic、Anthropic Messages、Antigravity、Claude CLI、Codex CLI、GitHub Copilot）
- **工具插件** — Web 工具（web_search、web_fetch）和消息工具迁移到插件系统
- **插件解析器** — 提供商工厂现支持插件优先解析，内置回退

#### 技术改进

- 添加 `SetPluginProviderResolver` 用于提供商插件集成
- 添加 `NewAgentLoopWithPluginTools` 用于工具插件合并
- 添加 `MergeFrom` 方法到 ToolRegistry 用于合并插件工具
- 添加 `InitializeToolsOnly` 到插件管理器用于早期工具初始化
- 更新渠道管理器使用插件系统进行初始化
- 移除渠道注册表中的遗留工厂模式代码

#### 变更文件

- `pkg/framework/` — 新包（7 个核心文件）
- `pkg/plugins/channels/` — 16 个渠道插件
- `pkg/plugins/providers/` — 8 个提供商插件
- `pkg/plugins/tools/` — 2 个工具插件
- `pkg/plugins/docs/` — 插件文档
- `cmd/moonhub/internal/gateway/helpers.go` — 插件导入和初始化
- `pkg/agent/loop.go` — 插件工具集成
- `pkg/channels/manager.go` — 基于插件的渠道初始化
- `pkg/providers/factory_provider.go` — 插件解析器支持
- `pkg/tools/registry.go` — MergeFrom 方法
- `docs/implementation/plugin-status.md` — 实现状态

</details>

<details>
<summary><strong>2025-03-19 — 自我进化系统实现</strong></summary>

#### 新功能

**自我进化行为模式检测系统**（`pkg/learning/`）

让 Agent 在每次交互中变得更好的全面学习系统：

- **模式检测器** — 使用正则模式、语义关键词分析和对话流分析的多层信号检测。支持中英文反馈检测。
- **工具追踪器** — 追踪工具使用统计，包括成功率、用户接受/拒绝、耗时指标（平均、P50、P95）和偏好评分。
- **行为评分器** — 多维评分系统，衡量响应质量、工具效率、上下文相关性、纠正率和适应速度。
- **模式进化** — 基于艾宾浩斯的衰减算法、模式合并、陈旧模式修剪和矛盾检测。
- **主动建议** — 基于检测到的模式和行为趋势生成优化建议。
- **Agent 集成** — 与 Agent 循环无缝集成，在对话中自动学习。

#### 技术改进

- 为模式查询添加 FTS5 全文搜索，正确处理特殊字符转义
- 在整个持久层实现正确的 JSON 错误处理
- 修复工具使用追踪中的 SQL 参数不匹配
- 添加完善的单元测试（43 个测试，100% 通过率）
- 更新 golangci-lint 配置到 v2 格式

#### Bug 修复

- 修复 `GetAllPatterns()`、`GetPatternsByCategory()`、`GetToolUsagePatterns()` 返回 nil 而非空切片
- 修复 `deduplicateSignals()` 返回 nil 而非空切片
- 修复 `RecordUserAcceptance()` 未递增 `UserAcceptedCalls` 计数器
- 修复 `calculatePreference()` 使用错误指标（成功率而非接受率）
- 移除模式检测循环中的过早 `break` 语句以捕获所有匹配信号

#### 变更文件

- `pkg/learning/` — 新包（14 个文件，约 3000 行）
- `pkg/agent/loop.go` — Agent 循环中的学习集成
- `pkg/agent/context.go` — 学习上下文注入
- `pkg/agent/memory.go` — 记忆-学习桥接
- `pkg/config/config.go` — 学习配置选项
- `.golangci.yaml` — 更新到 v2 格式

</details>
