# MoonHub

> [!NOTE]
> **Acknowledgments**
>
> This project was inspired by the feature integration of [TinyClaw](https://github.com/wgtechlabs/tinyclaw) and the lightweight design of [PicoClaw](https://github.com/sipeed/picoclaw), and continues to develop in its own unique direction.

**Documentation Index** (plugins, learning, compactor, SHIELD, memory, delegation, etc.): [`docs/README.md`](docs/README.md).

**[中文文档](README_CN.md)**

## Introduction

> **A ready-to-use agent that's perfect for everyday users.**

MoonHub is a **local-first AI assistant** designed for edge computing — Simple, Fast, Secure.

We believe AI shouldn't be the exclusive tool of tech experts. MoonHub is built for **everyday users** — no technical background required, just plug and play. Currently adapted for Linux, it runs perfectly on Raspberry Pi, industrial gateways, and various embedded devices.

### Design Philosophy

| Principle | Description |
|-----------|-------------|
| **Simple** | Zero learning curve. Works out of the box, as easy as any home appliance. |
| **Fast** | Ultra-lightweight. <10MB memory, 1-second cold start, millisecond response. |
| **Secure** | Local-first. Your data never leaves the device — privacy entirely in your hands. |

### Core Features

**🤖 Agent Collaboration Engine**

You're the commander, and the Agent team serves you. Each Agent has its own responsibilities — they communicate with each other, collaborate proactively, and seek your approval at critical decision points. This isn't a simple Q&A bot, but an intelligent team that truly understands context and autonomously drives tasks forward.

**🎨 Dynamic UI Generation**

Say goodbye to traditional Agents that "only output text." MoonHub can generate visual interactive interfaces in real-time based on your needs — financial dashboards, task managers, data visualizations — everything adapts on demand. You describe the idea, the Agent builds it for you.

**📱 Dedicated Application**

Users interact with the device through a dedicated app. Currently provided as a **PWA** for quick installation and offline use; native **mobile apps** are coming soon to cover more platforms and use cases.

### Infinite Possibilities

MoonHub is deeply rooted in edge computing scenarios, maintaining extreme lightweight (<10MB memory) while delivering smooth experience and complete functionality. While the project has a clear core roadmap, its architecture is designed to fully support secondary development for diverse edge scenarios—whether smart agriculture, industrial IoT, intelligent retail, or home automation, you can rapidly build your own intelligent solutions on top of MoonHub.

| Scenario | Description |
|----------|-------------|
| **Smart Irrigation** | Connect soil moisture and weather sensors. The Agent dynamically adjusts irrigation strategies based on real-time data for precision water-saving agriculture. |
| **Industrial Monitoring** | Deploy in production workshops for real-time equipment status collection, predictive maintenance alerts, and visualized operation dashboards. |
| **Smart Retail** | Connect foot traffic counters and inventory sensors to automatically generate restocking suggestions and sales analysis reports for business decisions. |
| **Energy Management** | Interface with smart meters and solar inverters to optimize power consumption strategies in real-time and generate energy reports. |
| **Smart CRM** | Integrate customer data and communication records. AI analyzes customer profiles and automatically generates follow-up reminders and sales opportunity insights. |
| **Intelligent Ops** | Connect server and application monitoring data. AI identifies anomaly patterns, triggers automatic alerts, and generates fault diagnosis reports. |
| **Smart Security** | Interface with cameras and door/window sensors. AI detects abnormal behaviors, pushes real-time alerts, and generates security logs. |
| **Smart Aquaculture** | Connect water quality sensors and feeding equipment. Real-time monitoring of aquaculture environment with automatic feeding adjustment and growth analysis reports. |
| **Smart Classroom** | Connect attendance devices and interactive displays. Automatically record attendance and assist teachers in generating personalized learning reports. |
| **Smart E-commerce** | Interface with order, inventory, and logistics systems. AI analyzes sales trends and automatically generates restocking suggestions and marketing strategies. |

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

### Planned

- **PWA Frontend** — Full user-facing PWA for device discovery, pairing, and day-to-day interaction beyond the provisioning wizard
- **Dynamic UI Generation** — Real-time visual component generation based on user needs (dashboards, task managers, data visualizations)
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

</details>

## Changelog

<details>
<summary><strong>2026-03-22 — Device provisioning documentation flow</strong></summary>

#### Summary

Repository documentation now follows the same **package docs → implementation status** pattern as routing, delegation, and compactor: English `pkg/provisioning/docs/`, Chinese deep-dive in `docs/implementation/provisioning-status.md`, and cross-links from the doc index, web guide, and `CLAUDE.md`.

#### Documentation

- [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md) — Scope, source map, integration table (web API, launcher, frontend)
- [`pkg/provisioning/docs/CONFIG.md`](pkg/provisioning/docs/CONFIG.md) — Environment variables, `provisioning.json`, persisted keys, HTTP/SSE and browser token notes
- [`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md) — Reading-order header linking the above; existing API and UI reference retained
- [`docs/README.md`](docs/README.md) — Subsystem table, first-time reading step 6, `docs/implementation/` row, `pkg/` overview entry
- [`web/README.md`](web/README.md) — Optional provisioning subsection (paths + doc flow)
- [`CLAUDE.md`](CLAUDE.md) — `provisioning/` package note, launcher env vars, doc links
- [`README.md`](README.md) / [`README_CN.md`](README_CN.md) — Device Provisioning listed under Implemented with doc links

</details>

<details>
<summary><strong>2026-03-21 — Smart Router V2 (4-Tier Model Routing)</strong></summary>

#### Summary

4-tier model routing system with rule-based classification, replacing the original 2-tier (light/heavy) system. Automatically selects the appropriate LLM based on message complexity.

#### New Features

**Smart Router V2** (`pkg/routing/`)
- **4-Tier Classification** — simple, moderate, complex, reasoning tiers with configurable boundaries
- **Rule-Based Scoring** — Sub-microsecond classification using structural features (no API calls)
- **Feature Extraction** — Token estimate, code blocks, tool calls, conversation depth, attachments
- **Attachment Hard Gate** — Multi-modal inputs automatically route to reasoning tier
- **Confidence Scoring** — Sigmoid-based confidence calculation for each classification
- **Signal Tracing** — Every decision includes explainable signals for debugging
- **Privacy-Safe Metrics** — Aggregate statistics without storing message content
- **Decision Recorder** — Ring buffer for recent decisions with tier filtering
- **HTTP Endpoints** — `/metrics`, `/routing/decisions`, `/routing/stats`
- **Backward Compatible** — Falls back to 2-tier mode when `light_model` is configured

#### Documentation

- [`pkg/routing/docs/README.md`](pkg/routing/docs/README.md) — Overview, architecture, quick start
- [`pkg/routing/docs/CONFIG.md`](pkg/routing/docs/CONFIG.md) — Configuration options, tier mapping, custom boundaries
- [`pkg/routing/docs/FEATURES.md`](pkg/routing/docs/FEATURES.md) — Feature extraction, scoring weights, examples
- [`pkg/routing/docs/METRICS.md`](pkg/routing/docs/METRICS.md) — Metrics collection, decision recorder, HTTP endpoints
- [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md) — Full implementation status
- [`docs/README.md`](docs/README.md) — Repository documentation index (updated)

#### Technical Details

- Score range: [-1.0, 1.0] with negative scores for simple messages
- Default boundaries: simple [-1, -0.05), moderate [-0.05, 0.15), complex [0.15, 0.35), reasoning [0.35, 1.0]
- Weights: short message (-0.10), code block (+0.40), long message (+0.35), attachment (1.0 hard gate)
- 42 unit tests, all passing

#### Files Changed

- `pkg/routing/` — Core implementation (tier, classifier, router, features, metrics, recorder)
- `pkg/routing/docs/` — New documentation directory (README, CONFIG, FEATURES, METRICS)
- `pkg/config/config.go` — RoutingConfig with TierMapping and TierBoundariesConfig
- `pkg/agent/instance.go` — RouterV2, TierCandidates fields and initialization
- `pkg/agent/loop.go` — Updated selectCandidates for 4-tier routing
- `pkg/health/server.go` — HTTP endpoints for metrics and decisions
- `docs/README.md` — Updated Smart Router section with full documentation links

</details>

<details>
<summary><strong>2026-03-21 — Delegation System (sub-agent orchestration)</strong></summary>

#### Summary

Sub-agent delegation with autonomous workflows: eight tools, SQLite store, session queue, background task injection in the agent loop, and `DelegationUserID` on tool execution context.

#### Documentation

- [`pkg/delegation/docs/README.md`](pkg/delegation/docs/README.md), [`pkg/delegation/docs/CONFIG.md`](pkg/delegation/docs/CONFIG.md) — Package-level docs (flow + config)
- [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md) — Deep implementation reference
- [`docs/README.md`](docs/README.md) — Repository index (delegation subsystem table)

#### Code (high level)

- `pkg/delegation/` — Core implementation
- `pkg/agent/delegation_integration.go`, `pkg/agent/instance.go`, `pkg/agent/loop.go` — Runtime wiring
- `pkg/config/config.go`, `pkg/config/defaults.go` — `DelegationConfig`

</details>

<details>
<summary><strong>2026-03-21 — SHIELD.md Anti-Malware Implementation</strong></summary>

#### New Features

**SHIELD.md Anti-Malware System** (`pkg/shield/`)

A runtime threat evaluation engine:

- **Threat Parser** — YAML-formatted SHIELD.md parser with support for threat definitions, directives, and metadata
- **Pattern Matcher** — Condition syntax support for tool calls, file paths, network egress, skill operations
- **Enforcement Actions** — Three action types: `block`, `require_approval`, `log` with priority-based resolution
- **Approval Workflow** — `/approve` and `/reject` commands for user-confirmed actions with 5-minute timeout
- **Tool Integration** — Shield evaluation integrated into `web_fetch` and `install_skill` tools
- **Default Threats** — 8 built-in threats covering SQL injection, command injection, path traversal, credential access, etc.

#### Technical Details

- Confidence threshold (0.85) with severity override for critical threats
- Action priority: `block` > `require_approval` > `log`
- Context-based approval bypass to prevent double-evaluation
- Comprehensive unit tests (31 tests, 100% pass rate)

#### Files Changed

- `pkg/shield/` — New package (11 core files + 5 test files)
- `pkg/agent/instance.go` — Shield and ApprovalManager initialization
- `pkg/agent/loop.go` — Shield evaluation in tool execution flow
- `pkg/commands/cmd_approve.go` — Approve/reject command handlers
- `pkg/tools/web.go` — Shield integration for network egress
- `pkg/tools/skills_install.go` — Shield integration for skill installation
- `docs/implementation/shield-status.md` — Implementation status

</details>

<details>
<summary><strong>2026-03-20 — Context Compactor &amp; documentation flow</strong></summary>

#### Features

- **Context Compactor** (`pkg/compactor/`) — Four-layer pipeline (rule-based pre-compression, deduplication, LLM summary, L0/L1/L2 tiers) integrated into Agent; see `compactor` config and [`pkg/compactor/docs/CONFIG.md`](pkg/compactor/docs/CONFIG.md).

#### Documentation

- Added repository documentation entry [`docs/README.md`](docs/README.md), distinguishing "package docs" from `docs/implementation/*-status.md` consistent with `pkg/learning/docs`.
- Added [`pkg/compactor/docs/`](pkg/compactor/docs/README.md) (README + CONFIG).
- Fixed link to `plugin-architecture-status.md` pointing to actual file [`docs/implementation/plugin-status.md`](docs/implementation/plugin-status.md).

</details>

<details>
<summary><strong>2025-03-20 — Plugin Architecture Implementation</strong></summary>

#### New Features

**Plugin Architecture System** (`pkg/framework/`, `pkg/plugins/`)

A comprehensive plugin system that makes channels, providers, and tools all extensible plugins:

- **Core Framework** — Plugin types, interfaces (Channel/Provider/Tool), registry system, and lifecycle manager
- **Channel Plugins** — 16 channel plugins migrated (Telegram, Discord, Slack, Matrix, Feishu, QQ, DingTalk, LINE, OneBot, WeCom, WeCom App, WeCom AIBot, Pico, IRC, MaixCam, WhatsApp)
- **Provider Plugins** — 8 provider plugins migrated (OpenAI Compat, OpenAI OAuth, Anthropic, Anthropic Messages, Antigravity, Claude CLI, Codex CLI, GitHub Copilot)
- **Tool Plugins** — Web tools (web_search, web_fetch) and message tool migrated to plugin system
- **Plugin Resolver** — Provider factory now supports plugin-first resolution with built-in fallback

#### Technical Improvements

- Added `SetPluginProviderResolver` for provider plugin integration
- Added `NewAgentLoopWithPluginTools` for tool plugin merging
- Added `MergeFrom` method to ToolRegistry for combining plugin tools
- Added `InitializeToolsOnly` to plugin manager for early tool initialization
- Updated channel manager to use plugin system for initialization
- Removed legacy factory pattern code from channel registry

#### Files Changed

- `pkg/framework/` — New package (7 core files)
- `pkg/plugins/channels/` — 16 channel plugins
- `pkg/plugins/providers/` — 8 provider plugins
- `pkg/plugins/tools/` — 2 tool plugins
- `pkg/plugins/docs/` — Plugin documentation
- `cmd/moonhub/internal/gateway/helpers.go` — Plugin imports and initialization
- `pkg/agent/loop.go` — Plugin tool integration
- `pkg/channels/manager.go` — Plugin-based channel initialization
- `pkg/providers/factory_provider.go` — Plugin resolver support
- `pkg/tools/registry.go` — MergeFrom method
- `docs/implementation/plugin-status.md` — Implementation status

</details>

<details>
<summary><strong>2025-03-19 — Self-Improving System Implementation</strong></summary>

#### New Features

**Self-Improving Behavioral Pattern Detection System** (`pkg/learning/`)

A comprehensive learning system that makes the agent better with every interaction:

- **Pattern Detector** — Multi-layer signal detection using regex patterns, semantic keyword analysis, and conversation flow analysis. Supports both English and Chinese feedback detection.
- **Tool Tracker** — Tracks tool usage statistics including success rates, user acceptance/rejection, duration metrics (avg, P50, P95), and preference scoring.
- **Behavioral Scorer** — Multi-dimensional scoring system measuring response quality, tool efficiency, context relevance, correction rate, and adaptation speed.
- **Pattern Evolution** — Ebbinghaus-inspired decay algorithm, pattern merging, pruning of stale patterns, and contradiction detection.
- **Proactive Suggestions** — Generates optimization suggestions based on detected patterns and behavioral trends.
- **Agent Integration** — Seamless integration with the agent loop for automatic learning during conversations.

#### Technical Improvements

- Added FTS5 full-text search for pattern queries with proper special character escaping
- Implemented proper JSON error handling throughout the persistence layer
- Fixed SQL parameter mismatches in tool usage tracking
- Added comprehensive unit tests (43 tests, 100% pass rate)
- Updated golangci-lint configuration to v2 format

#### Bug Fixes

- Fixed `GetAllPatterns()`, `GetPatternsByCategory()`, `GetToolUsagePatterns()` returning nil instead of empty slices
- Fixed `deduplicateSignals()` returning nil instead of empty slice
- Fixed `RecordUserAcceptance()` not incrementing `UserAcceptedCalls` counter
- Fixed `calculatePreference()` using wrong metric (success rate instead of acceptance rate)
- Removed premature `break` statements in pattern detection loops to capture all matching signals

#### Files Changed

- `pkg/learning/` — New package (14 files, ~3000 lines)
- `pkg/agent/loop.go` — Learning integration in agent loop
- `pkg/agent/context.go` — Learning context injection
- `pkg/agent/memory.go` — Memory-learning bridge
- `pkg/config/config.go` — Learning configuration options
- `.golangci.yaml` — Updated to v2 format

</details>
