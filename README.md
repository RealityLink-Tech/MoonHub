# MoonHub

> [!NOTE]
> **Acknowledgments**
>
> This project was inspired by the feature integration of [TinyClaw](https://github.com/warengonzaga/tinyclaw) and the lightweight design of [PicoClaw](https://github.com/sipeed/picoclaw), and continues to develop in its own unique direction.

**Documentation Index** (plugins, learning, compactor, SHIELD, memory, delegation, etc.): [`docs/README.md`](docs/README.md).

**Changelog**: [`CHANGELOG.md`](CHANGELOG.md) — Project updates and release notes

**[中文文档](README_CN.md)**

## Introduction

> **Your AI assistant — instant, alive, connected.**

MoonHub is an **AI assistant that works out of the box**. No technical background needed — plug in, pair, and start talking. Your data stays on your device. Your AI adapts to you.

### Design Philosophy

| Principle | Description |
|-----------|-------------|
| **Instant** | Zero learning curve. From power-on to conversation in 3 steps — no terminals, no configs, no technical background. |
| **Alive** | Interfaces that adapt to you. AI doesn't just output text — it builds dashboards, trackers, and tools on demand. Your idea is the blueprint. |
| **Connected** | Agents are social. Add friends, approve access, share selectively — like a social network for AI. |

### Core Features

**🤖 Agent Social Network**

Agents can add each other as friends — just like adding a contact in any messaging app. Send a friend request, get approved, and now your agents can talk. You decide what's shared and what stays private. Each Agent has a cryptographic identity, and communication is encrypted end-to-end. It's like building your own AI social circle.

- **Friend System** — Send requests, accept or reject, revoke when needed. Your agent only talks to agents you trust.
- **Zone-based Access Control** — Three-tier data architecture: **Private** (only you), **Shared** (friends can read), **Public** (anyone can read). You decide what each friend can see.
- **Local-first, Cloud-optional** — Agents on the same network connect directly. Need to reach someone across the internet? An optional cloud relay bridges the gap — no port forwarding required.

> [!IMPORTANT]
> The agent social network is **under active development**. Friend management and zone-based access control are implemented. Task delegation and file transfer between agents are defined in the protocol but not yet available in handlers. Stay tuned for updates.

**🎨 Dynamic UI Generation**

Say goodbye to traditional Agents that "only output text." MoonHub can generate visual interactive interfaces in real-time based on your needs — financial dashboards, task managers, data visualizations — everything adapts on demand. You describe the idea, the Agent builds it for you.

**📱 Dedicated Application**

Users interact with the device through a dedicated app. Currently provided as a **PWA** for quick installation and offline use; native **mobile apps** are coming soon to cover more platforms and use cases.

### What Can You Do?

MoonHub adapts to your life, not the other way around. Here are some ways people use it:

| Scenario | Description |
|----------|-------------|
| **Personal Productivity** | Manage schedules, track habits, organize notes. AI remembers your preferences and gets better over time. |
| **Smart Home** | Connect your devices — control lights, AC, curtains with a single message. AI learns your daily routines. |
| **Learning Companion** | Homework help, language practice, knowledge Q&A. Adaptive memory tracks your learning progress. |
| **Creative Workspace** | Describe an idea, AI builds dashboards, charts, and management tools for you. Need it? Create it. |
| **Team Collaboration** | Multiple Agents, each with a role — one gathers info, another analyzes data — all working together for you. |
| **Remote Manager** | Check home status, receive alerts, and control devices from anywhere through your phone. |

Your imagination is MoonHub's only boundary.

### Quick Start

1. **Power On** — Device automatically creates a WiFi hotspot (`MoonHub-XXXX`)
2. **Configure Network** — Connect to the hotspot, access the setup page, configure WiFi and set authorization code
3. **Install PWA** — After configuration, follow the guide to install the PWA app
4. **Start Using** — PWA automatically scans for local devices, enter the authorization code to start chatting, managing, and configuring

## Features

### Implemented

- **Adaptive Memory** — 3-layer memory system (episodic, semantic FTS5, temporal decay) that learns what to remember and forget over time.
- **Self-Improving** — Behavioral pattern detection system that learns from user feedback, tracks tool usage preferences, and evolves patterns over time.
- **Plugin Architecture** — Channels, providers, and tools are all plugins. The core stays tiny — everything else is extensible.
- **Context Compactor** — 4-layer context compaction pipeline (rules, dedup, LLM summary, L0/L1/L2 tiers) integrated in the agent loop; see [`docs/implementation/compactor-status.md`](docs/implementation/compactor-status.md) and [`pkg/compactor/docs/`](pkg/compactor/docs/README.md).
- **SHIELD.md Anti-Malware** — Runtime threat evaluation engine with YAML threat parsing, pattern matching, approval workflow, and 8 built-in threats; see [`docs/implementation/shield-status.md`](docs/implementation/shield-status.md) and [`pkg/shield/docs/`](pkg/shield/docs/README.md).
- **Delegation System** — Sub-agent orchestration (non-blocking and background tasks, template reuse, adaptive timeouts, SQLite persistence, Intercom pub/sub). Opt-in via `delegation.enabled` in `config.json` (default off); see [`pkg/delegation/docs/README.md`](pkg/delegation/docs/README.md), [`pkg/delegation/docs/CONFIG.md`](pkg/delegation/docs/CONFIG.md), and [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md).
- **Inter-Agent Comms (Intercom)** — In-process pub/sub for delegation-time signals: subscribe per topic (`On`), catch-all via `OnAny`, bounded per-topic retention with `Recent` / `RecentAll`; see [`pkg/delegation/intercom.go`](pkg/delegation/intercom.go) and the Intercom section in [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md).
- **Smart Router V2** — 4-tier model routing system (simple/moderate/complex/reasoning) with rule-based scoring, feature extraction, and privacy-safe metrics. Routes simple queries to cheap models and complex ones to powerful models; see [`pkg/routing/docs/README.md`](pkg/routing/docs/README.md) and [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md).
- **Device Provisioning** — Zero-config WiFi setup (hotspot, scan/connect, diagnostics, automatic and manual recovery, factory reset, auth code, SSE). Opt-in on the web launcher via `MOONHUB_PROVISIONING_ENABLED=1`; includes React provisioning wizard and optional PWA offline cache for that flow. See [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md), [`pkg/provisioning/docs/CONFIG.md`](pkg/provisioning/docs/CONFIG.md), and [`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md).
- **Cloud directory & relay** — Optional HTTP directory (PostgreSQL + Redis) for agent registration and online relay endpoints, plus a WebSocket relay with directory-backed Ed25519 auth; device-side [`pkg/transport`](pkg/transport/cloud.go) `CloudClient` and `Resolver` (LAN-first, then cloud). See [`docs/implementation/cloud-directory-relay-status.md`](docs/implementation/cloud-directory-relay-status.md), [`cloud/directory/docs/README.md`](cloud/directory/docs/README.md), [`cloud/relay/docs/README.md`](cloud/relay/docs/README.md), and [`pkg/transport/docs/README.md`](pkg/transport/docs/README.md).
- **MoonHub PWA (companion app)** — Installable progressive web app for LAN discovery, pairing, chat, Space, and settings; uses backend APIs such as `GET /api/discover`, `GET /api/devices`, and channel CRUD on `/api/channels`. Backend contract: [`web/backend/api/README.md`](web/backend/api/README.md). Frontend docs (split layout): `MoonHub-PWA/docs/README.md`.
- **Dynamic tools (AI-generated UI)** — Schema-driven tools persisted in SQLite (`dynamic_tools.db`), generated via LLM from natural language, executed server-side with optional HTTP fetch injection, exposed as LAN `/api/dynamic-tools`; PWA renders with **DynamicRenderer** and chat/space dynamic components. See [`pkg/dynamictools/docs/README.md`](pkg/dynamictools/docs/README.md), [`docs/implementation/dynamic-tools-status.md`](docs/implementation/dynamic-tools-status.md), and [`web/backend/api/README.md`](web/backend/api/README.md).
- **Agent Social Network** — Friend management (request/accept/reject/revoke) with Ed25519 cryptographic identity, MHP (MoonHub Protocol) envelope messaging, LAN-direct and cloud-relay transport, and 3-tier zone-based access control (Privacy/Shared/Public). See [`pkg/friends/`](pkg/friends/), [`pkg/protocol/mhp/`](pkg/protocol/mhp/), [`pkg/zones/`](pkg/zones/), and [`pkg/transport/`](pkg/transport/).

### Planned

- **Wasm tool engine** — Execute `engine: wasm` dynamic tools (wazero or equivalent); schema path is already shipped
- **Cross-agent task delegation** — Delegate tasks to friend agents via MHP protocol (types defined, handlers in progress)
- **Cross-agent file transfer** — Send files between friend agents with zone-based access checks (types defined, handlers in progress)
- **Native APP** — Native mobile applications for iOS and Android platforms

<details>
<summary><strong>Completed (click to expand)</strong></summary>

- ~~**Self-Improving** — Behavioral pattern detection that makes the agent better with every interaction. It grows with you.~~ → **Implemented**
- ~~**Plugin Architecture** — Channels, providers, and tools are all plugins. The core stays tiny — everything else is extensible.~~ → **Implemented**
- ~~**Context Compactor** — 4-layer context compaction pipeline with rule-based pre-compression, deduplication, LLM summarization, and tiered summaries.~~ → **Implemented**
- ~~**SHIELD.md Anti-Malware** — Runtime SHIELD.md enforcement engine with threat parsing, pattern matching, and built-in anti-malware protection.~~ → **Implemented**
- ~~**Delegation System** — Autonomous sub-agent orchestration with self-improving role templates, blackboard collaboration, and adaptive timeouts.~~ → **Implemented** (opt-in; see [`pkg/delegation/docs/`](pkg/delegation/docs/README.md) and [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md))
- ~~**Smart Routing** — 4-tier query classifier that routes simple queries to cheap models and complex ones to powerful ones, cutting LLM costs.~~ → **Implemented** (see [`pkg/routing/docs/`](pkg/routing/docs/README.md) and [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md))
- ~~**Inter-Agent Comms** — Lightweight pub/sub event bus for real-time inter-agent communication with wildcard subscriptions and bounded history.~~ → **Implemented** (delegation **Intercom** in [`pkg/delegation/intercom.go`](pkg/delegation/intercom.go); enabled with delegation)
- ~~**Device Provisioning** — Zero-config WiFi setup, recovery, factory reset, provisioning UI.~~ → **Implemented** (launcher opt-in; see [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md) and [`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md))
- ~~**PWA Frontend** — Companion PWA for discovery, pairing, chat, Space, and settings.~~ → **Implemented** (LAN APIs documented in [`web/backend/api/README.md`](web/backend/api/README.md); app repo `MoonHub-PWA`.)
- ~~**Dynamic UI Generation** — Real-time visual components from AI (dashboards, Space, chat cards).~~ → **Implemented** (schema phase: [`pkg/dynamictools/docs/README.md`](pkg/dynamictools/docs/README.md); Wasm execution still planned.)

</details>
