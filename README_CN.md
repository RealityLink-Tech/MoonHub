# MoonHub

> [!NOTE]
> **致谢**
>
> 本项目灵感来源于 [TinyClaw](https://github.com/warengonzaga/tinyclaw) 的功能集成和 [PicoClaw](https://github.com/sipeed/picoclaw) 的轻量化设计，在此基础上继续发展独属于本项目的方向。

**文档索引**（插件、学习、压缩器、SHIELD、记忆、委托、配网等）：[`docs/README.md`](docs/README.md)。

**更新日志**：[`CHANGELOG.md`](CHANGELOG.md) — 项目更新与发布说明

**配套客户端**：[`../MoonHub-PWA/README.md`](../MoonHub-PWA/README.md) — 用于配对、对话、Space 与设置的可安装 PWA

**[English](README.md)**

## 简介

> **你的 AI 助手——开箱即用，鲜活灵动，彼此连接。**

MoonHub 是一款**开箱即用的 AI 助手**。无需技术背景——插电、配对、开始对话。你的数据留在你的设备上，你的 AI 越来越懂你。

### 设计理念

| 原则 | 描述 |
|------|------|
| **Instant** | 零学习曲线。从上电到对话只需 3 步，无需终端、无需配置、无需技术背景。 |
| **Alive** | 界面随需而变。AI 不只输出文字，还能实时生成交互界面，你的需求就是它的蓝图。 |
| **Connected** | Agent 也有社交圈。加好友、审批访问、选择性共享——像社交软件一样连接 AI。 |

### 核心特性

**🤖 Agent 社交网络**

Agent 之间可以互相添加好友——就像在聊天软件里加联系人一样。发送好友请求、通过审批，你的 Agent 就可以和对方通信了。你能决定哪些数据可以共享、哪些保持私密。每个 Agent 都有加密身份，通信端到端加密。就像在搭建属于你自己的 AI 社交圈。

- **好友系统** — 发送请求、接受或拒绝、随时撤销。你的 Agent 只和你信任的 Agent 对话。
- **三层数据架构** — 数据按访问权限分为三层：**私密**（仅自己可见）、**共享**（好友可读）、**公开**（任何人可读）。你来决定每个好友能看什么。
- **局域网优先，云端可选** — 同一网络内的 Agent 直连互通。需要跨网络通信？可选的云端中继帮你桥接，无需公网 IP。

> [!IMPORTANT]
> Agent 社交网络功能**正在积极开发中**。好友管理和三层数据权限控制已实现。Agent 之间的任务委托和文件传输功能已在协议中定义，处理程序尚在开发。请关注后续更新。

**🎨 动态 UI 生成**

告别"只能输出文字"的传统 Agent。MoonHub 能够根据你的需求，实时生成可视化交互界面——财务仪表盘、任务管理器、数据看板，一切随需而变。你描述想法，Agent 为你构建。

**📱 专属应用**

用户通过专属应用与设备交互。目前以 **PWA** 形式提供，支持快速安装、离线使用；后续将推出原生 **APP**，覆盖更多平台与使用场景。前端仓库见 [`../MoonHub-PWA/README.md`](../MoonHub-PWA/README.md)。

### MoonHub 能做什么？

MoonHub 适应你的生活，而不是让你适应它。以下是一些常见的使用方式：

| 场景 | 描述 |
|------|------|
| **个人效率** | 管理日程、追踪习惯、整理笔记。AI 记住你的偏好，越用越顺手。 |
| **家庭智能** | 连接家中设备，一句话控制灯光、空调、窗帘。AI 学习你的生活习惯。 |
| **学习伴侣** | 辅导功课、语言练习、知识问答。自适应记忆追踪学习进度。 |
| **创意工作台** | 描述想法，AI 为你生成看板、图表、管理工具。需要什么，创造什么。 |
| **团队协作** | 多 Agent 各司其职，一个负责信息收集，一个负责数据分析，为你协同工作。 |
| **远程管家** | 出门在外，通过手机查看家中状态、接收告警、远程控制设备。 |

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
- **MoonHub PWA（配套客户端）** — 可安装的渐进式 Web 应用，用于局域网发现、配对、对话、Space 与设置；调用后端如 `GET /api/discover`、`GET /api/devices` 及 `/api/channels` 等频道 CRUD。后端契约见 [`web/backend/api/README.md`](web/backend/api/README.md)；前端文档（分仓布局）见 `MoonHub-PWA/docs/README.md`。
- **动态工具（AI 生成 UI）** — 基于 Schema 的动态工具，SQLite 持久化（`dynamic_tools.db`），自然语言经 LLM 生成或按内容哈希去重，服务端可选 HTTP 拉数并注入 schema，局域网 `/api/dynamic-tools`；PWA 通过 **DynamicRenderer** 与 chat/space 动态组件集渲染。参见 [`pkg/dynamictools/docs/README.md`](pkg/dynamictools/docs/README.md)、[`docs/implementation/dynamic-tools-status.md`](docs/implementation/dynamic-tools-status.md)、[`web/backend/api/README.md`](web/backend/api/README.md)。
- **Agent 社交网络** — 好友管理（请求/接受/拒绝/撤销），Ed25519 加密身份，MHP（MoonHub Protocol）信封消息，局域网直连与云端中继传输，三层数据权限控制（私密/共享/公开）。参见 [`pkg/friends/`](pkg/friends/)、[`pkg/protocol/mhp/`](pkg/protocol/mhp/)、[`pkg/zones/`](pkg/zones/)、[`pkg/transport/`](pkg/transport/)。

### 计划中

- **Wasm 工具引擎** — 执行 `engine: wasm` 的动态工具（如 wazero）；Schema 路径已交付
- **跨 Agent 任务委托** — 通过 MHP 协议向好友 Agent 委托任务（类型已定义，处理程序开发中）
- **跨 Agent 文件传输** — 在好友 Agent 之间传输文件，带数据权限检查（类型已定义，处理程序开发中）
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
- ~~**PWA 前端** — 配套 PWA：发现、配对、对话、Space 与设置。~~ → **已实现**（局域网 API 见 [`web/backend/api/README.md`](web/backend/api/README.md)；应用仓库 `MoonHub-PWA`。）
- ~~**动态 UI 生成** — AI 实时生成可视化组件（仪表盘、Space、对话卡片）。~~ → **已实现**（Schema 阶段：[`pkg/dynamictools/docs/README.md`](pkg/dynamictools/docs/README.md)；Wasm 执行仍在计划。）

</details>
