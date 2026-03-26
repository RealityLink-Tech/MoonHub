# MoonHub

> [!NOTE]
> **致谢**
>
> 本项目灵感来源于 [TinyClaw](https://github.com/warengonzaga/tinyclaw) 的功能集成和 [PicoClaw](https://github.com/sipeed/picoclaw) 的轻量化设计，在此基础上继续发展独属于本项目的方向。

**文档索引**（插件、学习、压缩器、SHIELD、记忆、委托、配网等）：[`docs/README.md`](docs/README.md)。

**更新日志**：[`CHANGELOG.md`](CHANGELOG.md) — 项目更新与发布说明

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
- **云目录与中继** — 可选 HTTP 目录（PostgreSQL + Redis）用于代理注册与中继端点，以及带目录侧 Ed25519 鉴权的 WebSocket 中继；设备侧 [`pkg/transport`](pkg/transport/cloud.go) 提供 `CloudClient` 与 `Resolver`（优先局域网、其次云端）。参见 [`docs/implementation/cloud-directory-relay-status.md`](docs/implementation/cloud-directory-relay-status.md)、[`cloud/directory/docs/README.md`](cloud/directory/docs/README.md)、[`cloud/relay/docs/README.md`](cloud/relay/docs/README.md)、[`pkg/transport/docs/README.md`](pkg/transport/docs/README.md)。

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
