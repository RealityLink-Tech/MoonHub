# Changelog

All notable changes to MoonHub will be documented in this file.

---

## 2026-03-26 — Friends System & Identity Management

### Summary

Added cryptographic agent identity with Ed25519 keypairs, friend request lifecycle management, and SQLite-backed device friend store.

### New Features

**Agent Identity** (`pkg/agentidentity/`)
- Ed25519 keypair generation for cryptographically secure agent identities
- AgentID derivation from public key with stable base32 encoding
- Cross-platform identity verification and signing

**Friends System** (`pkg/friends/`)
- Friend request lifecycle: request → accept/reject → revoke
- SQLite-backed friend store with full CRUD operations
- Per-friend trust levels and capabilities

**Zone-Based Access Control** (`pkg/zones/`)
- Data partition management with zone-based access control
- Agent collaboration isolation zones

**Protocol MHP** (`pkg/protocol/mhp/`)
- Signed envelope with Ed25519 signing and verification
- Replay protection for message integrity
- Message types for friend, task, file, and discovery operations

### Files Changed

- `pkg/agentidentity/` — New package (identity.go, identity_test.go)
- `pkg/friends/` — New package (manager.go, store.go, tests)
- `pkg/zones/` — New package (zone.go, zone_test.go)
- `pkg/protocol/mhp/` — New package (envelope.go, types.go, tests)

---

## 2026-03-25 — LAN Discovery & Pairing System

### Summary

Local network device discovery via mDNS and secure pairing with authorization codes for PWA clients.

### New Features

**mDNS Service Discovery** (`pkg/mdns/`)
- Broadcast `_moonhub._tcp.local.` service
- TXT records: id, name, version, port
- IPv4/IPv6 support with stable device IDs

**Pairing System** (`pkg/devices/`)
- 2-letter + 4-digit auth codes (5-min expiry, one-time use)
- 32-byte hex tokens (30-day validity)
- Persistent device storage

**LAN Session Management** (`pkg/social/`)
- Per-device session isolation
- Automatic cleanup of inactive sessions

**API Endpoints** (`web/backend/api/`)
- `/api/ping`, `/api/system/info` — device status
- `/api/auth/status`, `/api/auth/pair`, `/api/auth/verify` — pairing flow
- Private IP range validation for security

### Documentation

- [`pkg/mdns/README.md`](pkg/mdns/README.md) — mDNS service discovery overview
- [`pkg/mdns/docs/CONFIG.md`](pkg/mdns/docs/CONFIG.md) — mDNS configuration options
- [`pkg/devices/README.md`](pkg/devices/README.md) — Device pairing system overview
- [`pkg/devices/docs/CONFIG.md`](pkg/devices/docs/CONFIG.md) — Pairing configuration options
- [`docs/implementation/lan-discovery-status.md`](docs/implementation/lan-discovery-status.md) — LAN discovery implementation status
- [`docs/implementation/lan-pairing-status.md`](docs/implementation/lan-pairing-status.md) — Pairing implementation status
- [`web/backend/api/README.md`](web/backend/api/README.md) — API endpoints documentation

### Technical Details

- Stable device ID generation via MAC address hashing
- One-time auth code consumption
- Token-based session authentication
- Private IP range validation (10.x, 172.16-31.x, 192.168.x)
- 34 files changed, 4698 lines added

---

## 2026-03-22 — Device provisioning documentation flow

### Summary

Repository documentation now follows the same **package docs → implementation status** pattern as routing, delegation, and compactor: English `pkg/provisioning/docs/`, Chinese deep-dive in `docs/implementation/provisioning-status.md`, and cross-links from the doc index, web guide, and `CLAUDE.md`.

### Documentation

- [`pkg/provisioning/docs/README.md`](pkg/provisioning/docs/README.md) — Scope, source map, integration table (web API, launcher, frontend)
- [`pkg/provisioning/docs/CONFIG.md`](pkg/provisioning/docs/CONFIG.md) — Environment variables, `provisioning.json`, persisted keys, HTTP/SSE and browser token notes
- [`docs/implementation/provisioning-status.md`](docs/implementation/provisioning-status.md) — Reading-order header linking the above; existing API and UI reference retained
- [`docs/README.md`](docs/README.md) — Subsystem table, first-time reading step 6, `docs/implementation/` row, `pkg/` overview entry
- [`web/README.md`](web/README.md) — Optional provisioning subsection (paths + doc flow)
- [`CLAUDE.md`](CLAUDE.md) — `provisioning/` package note, launcher env vars, doc links
- [`README.md`](README.md) / [`README_CN.md`](README_CN.md) — Device Provisioning listed under Implemented with doc links

---

## 2026-03-21 — Smart Router V2 (4-Tier Model Routing)

### Summary

4-tier model routing system with rule-based classification, replacing the original 2-tier (light/heavy) system. Automatically selects the appropriate LLM based on message complexity.

### New Features

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

### Documentation

- [`pkg/routing/docs/README.md`](pkg/routing/docs/README.md) — Overview, architecture, quick start
- [`pkg/routing/docs/CONFIG.md`](pkg/routing/docs/CONFIG.md) — Configuration options, tier mapping, custom boundaries
- [`pkg/routing/docs/FEATURES.md`](pkg/routing/docs/FEATURES.md) — Feature extraction, scoring weights, examples
- [`pkg/routing/docs/METRICS.md`](pkg/routing/docs/METRICS.md) — Metrics collection, decision recorder, HTTP endpoints
- [`docs/implementation/routing-status.md`](docs/implementation/routing-status.md) — Full implementation status
- [`docs/README.md`](docs/README.md) — Repository documentation index (updated)

### Technical Details

- Score range: [-1.0, 1.0] with negative scores for simple messages
- Default boundaries: simple [-1, -0.05), moderate [-0.05, 0.15), complex [0.15, 0.35), reasoning [0.35, 1.0]
- Weights: short message (-0.10), code block (+0.40), long message (+0.35), attachment (1.0 hard gate)
- 42 unit tests, all passing

### Files Changed

- `pkg/routing/` — Core implementation (tier, classifier, router, features, metrics, recorder)
- `pkg/routing/docs/` — New documentation directory (README, CONFIG, FEATURES, METRICS)
- `pkg/config/config.go` — RoutingConfig with TierMapping and TierBoundariesConfig
- `pkg/agent/instance.go` — RouterV2, TierCandidates fields and initialization
- `pkg/agent/loop.go` — Updated selectCandidates for 4-tier routing
- `pkg/health/server.go` — HTTP endpoints for metrics and decisions
- `docs/README.md` — Updated Smart Router section with full documentation links

---

## 2026-03-21 — Delegation System (sub-agent orchestration)

### Summary

Sub-agent delegation with autonomous workflows: eight tools, SQLite store, session queue, background task injection in the agent loop, and `DelegationUserID` on tool execution context.

### Documentation

- [`pkg/delegation/docs/README.md`](pkg/delegation/docs/README.md), [`pkg/delegation/docs/CONFIG.md`](pkg/delegation/docs/CONFIG.md) — Package-level docs (flow + config)
- [`docs/implementation/delegation-status.md`](docs/implementation/delegation-status.md) — Deep implementation reference
- [`docs/README.md`](docs/README.md) — Repository index (delegation subsystem table)

### Code (high level)

- `pkg/delegation/` — Core implementation
- `pkg/agent/delegation_integration.go`, `pkg/agent/instance.go`, `pkg/agent/loop.go` — Runtime wiring
- `pkg/config/config.go`, `pkg/config/defaults.go` — `DelegationConfig`

---

## 2026-03-21 — SHIELD.md Anti-Malware Implementation

### New Features

**SHIELD.md Anti-Malware System** (`pkg/shield/`)

A runtime threat evaluation engine:

- **Threat Parser** — YAML-formatted SHIELD.md parser with support for threat definitions, directives, and metadata
- **Pattern Matcher** — Condition syntax support for tool calls, file paths, network egress, skill operations
- **Enforcement Actions** — Three action types: `block`, `require_approval`, `log` with priority-based resolution
- **Approval Workflow** — `/approve` and `/reject` commands for user-confirmed actions with 5-minute timeout
- **Tool Integration** — Shield evaluation integrated into `web_fetch` and `install_skill` tools
- **Default Threats** — 8 built-in threats covering SQL injection, command injection, path traversal, credential access, etc.

### Technical Details

- Confidence threshold (0.85) with severity override for critical threats
- Action priority: `block` > `require_approval` > `log`
- Context-based approval bypass to prevent double-evaluation
- Comprehensive unit tests (31 tests, 100% pass rate)

### Files Changed

- `pkg/shield/` — New package (11 core files + 5 test files)
- `pkg/agent/instance.go` — Shield and ApprovalManager initialization
- `pkg/agent/loop.go` — Shield evaluation in tool execution flow
- `pkg/commands/cmd_approve.go` — Approve/reject command handlers
- `pkg/tools/web.go` — Shield integration for network egress
- `pkg/tools/skills_install.go` — Shield integration for skill installation
- `docs/implementation/shield-status.md` — Implementation status

---

## 2026-03-20 — Context Compactor & documentation flow

### Features

- **Context Compactor** (`pkg/compactor/`) — Four-layer pipeline (rule-based pre-compression, deduplication, LLM summary, L0/L1/L2 tiers) integrated into Agent; see `compactor` config and [`pkg/compactor/docs/CONFIG.md`](pkg/compactor/docs/CONFIG.md).

### Documentation

- Added repository documentation entry [`docs/README.md`](docs/README.md), distinguishing "package docs" from `docs/implementation/*-status.md` consistent with `pkg/learning/docs`.
- Added [`pkg/compactor/docs/`](pkg/compactor/docs/README.md) (README + CONFIG).
- Fixed link to `plugin-architecture-status.md` pointing to actual file [`docs/implementation/plugin-status.md`](docs/implementation/plugin-status.md).

---

## 2025-03-20 — Plugin Architecture Implementation

### New Features

**Plugin Architecture System** (`pkg/framework/`, `pkg/plugins/`)

A comprehensive plugin system that makes channels, providers, and tools all extensible plugins:

- **Core Framework** — Plugin types, interfaces (Channel/Provider/Tool), registry system, and lifecycle manager
- **Channel Plugins** — 16 channel plugins migrated (Telegram, Discord, Slack, Matrix, Feishu, QQ, DingTalk, LINE, OneBot, WeCom, WeCom App, WeCom AIBot, Pico, IRC, MaixCam, WhatsApp)
- **Provider Plugins** — 8 provider plugins migrated (OpenAI Compat, OpenAI OAuth, Anthropic, Anthropic Messages, Antigravity, Claude CLI, Codex CLI, GitHub Copilot)
- **Tool Plugins** — Web tools (web_search, web_fetch) and message tool migrated to plugin system
- **Plugin Resolver** — Provider factory now supports plugin-first resolution with built-in fallback

### Technical Improvements

- Added `SetPluginProviderResolver` for provider plugin integration
- Added `NewAgentLoopWithPluginTools` for tool plugin merging
- Added `MergeFrom` method to ToolRegistry for combining plugin tools
- Added `InitializeToolsOnly` to plugin manager for early tool initialization
- Updated channel manager to use plugin system for initialization
- Removed legacy factory pattern code from channel registry

### Files Changed

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

---

## 2025-03-19 — Self-Improving System Implementation

### New Features

**Self-Improving Behavioral Pattern Detection System** (`pkg/learning/`)

A comprehensive learning system that makes the agent better with every interaction:

- **Pattern Detector** — Multi-layer signal detection using regex patterns, semantic keyword analysis, and conversation flow analysis. Supports both English and Chinese feedback detection.
- **Tool Tracker** — Tracks tool usage statistics including success rates, user acceptance/rejection, duration metrics (avg, P50, P95), and preference scoring.
- **Behavioral Scorer** — Multi-dimensional scoring system measuring response quality, tool efficiency, context relevance, correction rate, and adaptation speed.
- **Pattern Evolution** — Ebbinghaus-inspired decay algorithm, pattern merging, pruning of stale patterns, and contradiction detection.
- **Proactive Suggestions** — Generates optimization suggestions based on detected patterns and behavioral trends.
- **Agent Integration** — Seamless integration with the agent loop for automatic learning during conversations.

### Technical Improvements

- Added FTS5 full-text search for pattern queries with proper special character escaping
- Implemented proper JSON error handling throughout the persistence layer
- Fixed SQL parameter mismatches in tool usage tracking
- Added comprehensive unit tests (43 tests, 100% pass rate)
- Updated golangci-lint configuration to v2 format

### Bug Fixes

- Fixed `GetAllPatterns()`, `GetPatternsByCategory()`, `GetToolUsagePatterns()` returning nil instead of empty slices
- Fixed `deduplicateSignals()` returning nil instead of empty slice
- Fixed `RecordUserAcceptance()` not incrementing `UserAcceptedCalls` counter
- Fixed `calculatePreference()` using wrong metric (success rate instead of acceptance rate)
- Removed premature `break` statements in pattern detection loops to capture all matching patterns

### Files Changed

- `pkg/learning/` — New package (14 files, ~3000 lines)
- `pkg/agent/loop.go` — Learning integration in agent loop
- `pkg/agent/context.go` — Learning context injection
- `pkg/agent/memory.go` — Memory-learning bridge
- `pkg/config/config.go` — Learning configuration options
- `.golangci.yaml` — Updated to v2 format
