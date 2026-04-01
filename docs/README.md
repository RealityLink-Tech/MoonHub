# MoonHub Documentation Index

This page is the **entry point and reading guide** for the repository documentation.

## Recommended Reading Order (First Time)

1. [Repository Root README](../README.md) — Feature overview, license, and recent changelog summary
2. [CHANGELOG.md](../CHANGELOG.md) — Project updates and release notes
3. [CLAUDE.md](../CLAUDE.md) — Project architecture, build commands, and development guidelines for Claude Code
4. [MoonHub-PWA README](../../MoonHub-PWA/README.md) — companion app overview for pairing, chat, Space, and settings in the split-repo layout
5. [Troubleshooting](./troubleshooting.md), [Debug Guide](./debug.md) — Runtime issues reference
6. [Tools & Capabilities Configuration](./tools_configuration.md) — Tool-side configuration (`tools.*` in `config.json`)
7. If you enable **sub-agent delegation** (`delegation.enabled`): [pkg/delegation/docs/README.md](../pkg/delegation/docs/README.md) → [CONFIG.md](../pkg/delegation/docs/CONFIG.md) → [implementation status](./implementation/delegation-status.md)
8. If you enable **device provisioning** on the web launcher (`MOONHUB_PROVISIONING_ENABLED=1`): [pkg/provisioning/docs/README.md](../pkg/provisioning/docs/README.md) → [CONFIG.md](../pkg/provisioning/docs/CONFIG.md) → [implementation status](./implementation/provisioning-status.md) (detailed reference)
9. If you deploy **cloud directory + relay** (optional Phase 3 services): [cloud/directory/docs/README.md](../cloud/directory/docs/README.md) → [CONFIG.md](../cloud/directory/docs/CONFIG.md) → [cloud/relay/docs/README.md](../cloud/relay/docs/README.md) → [CONFIG.md](../cloud/relay/docs/CONFIG.md) → [pkg/transport/docs/README.md](../pkg/transport/docs/README.md) → [implementation status](./implementation/cloud-directory-relay-status.md)
10. If you integrate or debug the **companion PWA** (LAN discovery, pairing, channel settings, chat): [web/backend/api/README.md](../web/backend/api/README.md) (authoritative HTTP contract) → [`../../MoonHub-PWA/docs/README.md`](../../MoonHub-PWA/docs/README.md) → [LAN discovery](./implementation/lan-discovery-status.md) / [LAN pairing](./implementation/lan-pairing-status.md) on the device side
11. If you work on **AI-generated dynamic UI** (Space home tools, schema execution): [pkg/dynamictools/docs/README.md](../pkg/dynamictools/docs/README.md) → [web/backend/api/README.md](../web/backend/api/README.md) ( `/api/dynamic-tools` ) → [implementation status](./implementation/dynamic-tools-status.md) → [`../../MoonHub-PWA/docs/ARCHITECTURE.md`](../../MoonHub-PWA/docs/ARCHITECTURE.md) / [`../../MoonHub-PWA/docs/services.md`](../../MoonHub-PWA/docs/services.md)

## Documentation Flow by Subsystem

### Plugin Architecture (Channel / Provider / Tool)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/framework/docs/README.md`](../pkg/framework/docs/README.md) | Framework positioning, package names, and import conventions |
| 2 | [`pkg/framework/docs/ARCHITECTURE.md`](../pkg/framework/docs/ARCHITECTURE.md) | Interfaces, Registry, lifecycle |
| 3 | [`pkg/framework/docs/INTEGRATION.md`](../pkg/framework/docs/INTEGRATION.md) | Integration with Gateway, agent, and subsystems |
| 4 | [`pkg/plugins/docs/README.md`](../pkg/plugins/docs/README.md) | Built-in plugin directory conventions |
| 5 | [`pkg/plugins/docs/PLUGIN_INDEX.md`](../pkg/plugins/docs/PLUGIN_INDEX.md) | Plugin list and paths |
| 6 | [`pkg/plugins/docs/ADDING_A_PLUGIN.md`](../pkg/plugins/docs/ADDING_A_PLUGIN.md) | Steps to add or migrate plugins |
| Status | [`docs/implementation/plugin-status.md`](./implementation/plugin-status.md) | Design decisions and implementation progress |

### Self-Improving System

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/learning/docs/README.md`](../pkg/learning/docs/README.md) | Feedback and capability overview |
| 2 | [`pkg/learning/docs/CONFIG.md`](../pkg/learning/docs/CONFIG.md) | `learning` configuration options |
| 3 | [`pkg/learning/docs/I18N.md`](../pkg/learning/docs/I18N.md) | Language and detection behavior |
| 4 | [`pkg/learning/docs/EXAMPLES.md`](../pkg/learning/docs/EXAMPLES.md), [`SupportedPatterns.md`](../pkg/learning/docs/SupportedPatterns.md) | Examples and pattern tables |
| Status | [`docs/implementation/learning-status.md`](./implementation/learning-status.md) | Implementation status and integration points |

### Context Compactor

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/compactor/docs/README.md`](../pkg/compactor/docs/README.md) | Four-layer pipeline and code entry points |
| 2 | [`pkg/compactor/docs/CONFIG.md`](../pkg/compactor/docs/CONFIG.md) | `compactor` configuration and environment variables |
| Status | [`docs/implementation/compactor-status.md`](./implementation/compactor-status.md) | Architecture, storage, agent integration, and test commands |

### SHIELD Runtime Security (`pkg/shield`)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/shield/docs/README.md`](../pkg/shield/docs/README.md) | Responsibilities, source map, and integration entry |
| 2 | [`pkg/shield/docs/SHIELD_MD.md`](../pkg/shield/docs/SHIELD_MD.md) | `SHIELD.md` YAML, conditional DSL, confidence semantics |
| 3 | [`pkg/shield/docs/EXAMPLES.md`](../pkg/shield/docs/EXAMPLES.md) | Workspace policy examples and approval commands |
| Status | [`docs/implementation/shield-status.md`](./implementation/shield-status.md) | Design, built-in threat tables, integration checklist, and tests |

### Adaptive Memory

| Document | Description |
| --- | --- |
| [`docs/implementation/memory-status.md`](./implementation/memory-status.md) | Package structure, capabilities, configuration, and integration notes; source in `pkg/adaptive_memory/` |

### Delegation System (sub-agent orchestration)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/delegation/docs/README.md`](../pkg/delegation/docs/README.md) | Responsibilities, source map, integration table |
| 2 | [`pkg/delegation/docs/CONFIG.md`](../pkg/delegation/docs/CONFIG.md) | `delegation` in `config.json` and environment variables |
| Status | [`docs/implementation/delegation-status.md`](./implementation/delegation-status.md) | Schema, eight tools, Intercom topics, examples, tests |

### Smart Router (4-Tier Model Routing)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/routing/docs/README.md`](../pkg/routing/docs/README.md) | Overview, architecture, quick start, and 4-tier system |
| 2 | [`pkg/routing/docs/CONFIG.md`](../pkg/routing/docs/CONFIG.md) | Configuration options, tier mapping, custom boundaries |
| 3 | [`pkg/routing/docs/FEATURES.md`](../pkg/routing/docs/FEATURES.md) | Feature extraction, scoring weights, examples |
| 4 | [`pkg/routing/docs/METRICS.md`](../pkg/routing/docs/METRICS.md) | Metrics collection, decision recorder, HTTP endpoints |
| Status | [`docs/implementation/routing-status.md`](./implementation/routing-status.md) | Full implementation status, integration points, tests |

### Device Provisioning (WiFi / hotspot / recovery)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/provisioning/docs/README.md`](../pkg/provisioning/docs/README.md) | Package scope, source map, integration entry points |
| 2 | [`pkg/provisioning/docs/CONFIG.md`](../pkg/provisioning/docs/CONFIG.md) | Environment variables, `device.*` / `onboarding.*` keys, sidecar `provisioning.json` |
| Status | [`docs/implementation/provisioning-status.md`](./implementation/provisioning-status.md) | Full implementation status (Chinese), API table, frontend layout, tests |

### LAN Communication (Phase 1)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/mdns/README.md`](../pkg/mdns/README.md) | mDNS service discovery overview |
| 2 | [`pkg/mdns/docs/CONFIG.md`](../pkg/mdns/docs/CONFIG.md) | mDNS configuration options |
| 3 | [`pkg/devices/README.md`](../pkg/devices/README.md) | Device pairing and token management |
| 4 | [`pkg/devices/docs/CONFIG.md`](../pkg/devices/docs/CONFIG.md) | Pairing configuration options |
| Status | [`docs/implementation/lan-discovery-status.md`](./implementation/lan-discovery-status.md) | Discovery implementation status |
| Status | [`docs/implementation/lan-pairing-status.md`](./implementation/lan-pairing-status.md) | Pairing implementation status |

### Cloud directory & relay (optional)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`cloud/directory/docs/README.md`](../cloud/directory/docs/README.md) | HTTP API, PostgreSQL store, Redis online cache |
| 2 | [`cloud/directory/docs/CONFIG.md`](../cloud/directory/docs/CONFIG.md) | `directory-service` flags, `DB_*`, `REDIS_URL` |
| 3 | [`cloud/relay/docs/README.md`](../cloud/relay/docs/README.md) | WebSocket bridge, Bearer + challenge auth, `CONNECT` handshake |
| 4 | [`cloud/relay/docs/CONFIG.md`](../cloud/relay/docs/CONFIG.md) | `relay` flags (`-addr`, `-directory-url`) |
| 5 | [`pkg/transport/docs/README.md`](../pkg/transport/docs/README.md) | `CloudClient`, `Resolver` (LAN first, then cloud), `Manager` |
| Status | [`docs/implementation/cloud-directory-relay-status.md`](./implementation/cloud-directory-relay-status.md) | Implementation status (Chinese) |

### Channels

Channel architecture, migration, and how to implement a channel: [`pkg/channels/README.md`](../pkg/channels/README.md). Per-channel behavior also lives with each plugin under [`pkg/plugins/channels/`](../pkg/plugins/channels/) (see [`pkg/plugins/docs/PLUGIN_INDEX.md`](../pkg/plugins/docs/PLUGIN_INDEX.md)).

### Web Interface

| Document | Description |
| --- | --- |
| [`web/README.md`](../web/README.md) | Web interface development (React + Vite frontend, Go backend); includes device provisioning UI and API notes |
| [`web/backend/api/README.md`](../web/backend/api/README.md) | **HTTP API reference** for the Go backend: LAN discovery (`/api/discover`), paired devices (`/api/devices`), auth, channel CRUD, **dynamic tools** (`/api/dynamic-tools`), config, chat, gateway, etc. Companion app docs: [`../../MoonHub-PWA/docs/README.md`](../../MoonHub-PWA/docs/README.md). |

### Dynamic tools (AI-generated UI)

| Order | Document | Description |
| --- | --- | --- |
| 1 | [`pkg/dynamictools/docs/README.md`](../pkg/dynamictools/docs/README.md) | Package scope, files, persistence path, integration with HTTP API and PWA |
| Status | [`docs/implementation/dynamic-tools-status.md`](./implementation/dynamic-tools-status.md) | Shipped vs roadmap (Wasm engine, ReadChannel, etc.) |

---

## `docs/implementation/` Overview

| File | Topic |
| --- | --- |
| [`plugin-status.md`](./implementation/plugin-status.md) | Plugin architecture implementation status |
| [`learning-status.md`](./implementation/learning-status.md) | Self-improving system implementation status |
| [`compactor-status.md`](./implementation/compactor-status.md) | Context compactor implementation status |
| [`shield-status.md`](./implementation/shield-status.md) | SHIELD runtime security implementation status |
| [`memory-status.md`](./implementation/memory-status.md) | Memory system implementation status |
| [`delegation-status.md`](./implementation/delegation-status.md) | Sub-agent delegation orchestration implementation status |
| [`routing-status.md`](./implementation/routing-status.md) | Smart Router V2 (4-tier routing) implementation status |
| [`provisioning-status.md`](./implementation/provisioning-status.md) | Device provisioning (WiFi, hotspot, SSE, recovery, auth code, web UI) implementation status |
| [`lan-discovery-status.md`](./implementation/lan-discovery-status.md) | LAN mDNS discovery implementation status |
| [`lan-pairing-status.md`](./implementation/lan-pairing-status.md) | LAN device pairing implementation status |
| [`cloud-directory-relay-status.md`](./implementation/cloud-directory-relay-status.md) | Cloud directory, WebSocket relay, and `pkg/transport` cloud path |
| [`dynamic-tools-status.md`](./implementation/dynamic-tools-status.md) | Dynamic tools: schema engine, SQLite, LAN HTTP API, PWA renderer |

---

## `docs/channels/` Overview

Per-channel specific documentation:

| Directory | Channel |
| --- | --- |
| [`telegram/`](./channels/telegram/) | Telegram channel documentation |
| [`discord/`](./channels/discord/) | Discord channel documentation |
| [`slack/`](./channels/slack/) | Slack channel documentation |
| [`matrix/`](./channels/matrix/) | Matrix channel documentation |
| [`qq/`](./channels/qq/) | QQ channel documentation |
| [`onebot/`](./channels/onebot/) | OneBot channel documentation |
| [`dingtalk/`](./channels/dingtalk/) | DingTalk channel documentation |
| [`feishu/`](./channels/feishu/) | Feishu/Lark channel documentation |
| [`wecom/`](./channels/wecom/) | WeCom (企业微信) channel documentation |
| [`line/`](./channels/line/) | LINE channel documentation |
| [`maixcam/`](./channels/maixcam/) | MaixCam channel documentation |

---

## Package Structure Overview

### Core Packages (`pkg/`)

| Package | Description |
| --- | --- |
| `agent/` | Main agent loop, message handling, tool execution |
| `channels/` | Platform integrations (Telegram, Discord, Slack, Matrix, QQ, WeChat, etc.) |
| `providers/` | LLM provider integrations (OpenAI, Anthropic, Gemini, Zhipu, DeepSeek, etc.) |
| `tools/` | Available tools (web search, file operations, cron, shell, MCP, etc.) |
| `skills/` | Extensible skill system for adding capabilities |
| `config/` | Configuration management and loading |
| `memory/` | Long-term memory system (MEMORY.md) |
| `session/` | Session and conversation management |
| `bus/` | Internal event bus for channel communication |
| `commands/` | Slash command definitions and execution |
| `routing/` | Smart Router V2 (4-tier model routing) |
| `mcp/` | Model Context Protocol integration |
| `auth/` | OAuth authentication management |
| `cron/` | Scheduled task management |
| `health/` | Health check and metrics server |
| `heartbeat/` | Periodic task prompts |
| `identity/` | Unified user identity management |
| `logger/` | Structured logging |
| `media/` | Media file lifecycle management |
| `migrate/` | Configuration migration utilities |
| `state/` | Persistent state management |
| `utils/` | Shared utilities |
| `voice/` | Voice/audio processing |
| `devices/` | Hardware device interfaces (I2C, SPI) |
| `provisioning/` | Device WiFi provisioning, hotspot, recovery, auth code (opt-in via launcher env) |
| `transport/` | Agent-to-agent connections: `Manager`, `Resolver` (LAN vs cloud), `CloudClient` for directory HTTP; see [pkg/transport/docs/README.md](../pkg/transport/docs/README.md) |
| `dynamictools/` | AI-generated dynamic tools: SQLite `ToolManager`, `SchemaEngine`, host HTTP fetch; LAN `/api/dynamic-tools`; see [pkg/dynamictools/docs/README.md](../pkg/dynamictools/docs/README.md) |
| `fileutil/` | File operation utilities |
| `constants/` | Shared constants |

### Optional cloud binaries (`cmd/`)

| Command | Description |
| --- | --- |
| `directory-service/` | HTTP agent directory (PostgreSQL + Redis); see [cloud/directory/docs/README.md](../cloud/directory/docs/README.md) |
| `relay/` | WebSocket relay with directory-backed auth; see [cloud/relay/docs/README.md](../cloud/relay/docs/README.md) |

### CLI Commands (`cmd/moonhub/internal/`)

| Command | Description |
| --- | --- |
| `agent/` | Interactive chat mode (`moonhub agent`) |
| `gateway/` | Long-running bot server for multi-channel support (`moonhub gateway`) |
| `onboard/` | Initial setup wizard (`moonhub onboard`) |
| `auth/` | OAuth authentication management (`moonhub auth`) |
| `cron/` | Scheduled task management (`moonhub cron`) |
| `skills/` | Skills management commands |
| `status/` | System status display |
| `version/` | Version information |
| `migrate/` | Configuration migration |
| `model/` | Model management |

### Web Interface (`web/`)

| Directory | Description |
| --- | --- |
| `frontend/` | React + Vite + TanStack Router dashboard |
| `backend/` | Go web server with embedded frontend |

---

## Supported LLM Providers (`pkg/providers/`)

MoonHub supports multiple LLM providers with a unified interface:

| Provider | Package | Models |
| --- | --- | --- |
| OpenAI | `openai_compat/` | GPT-4, GPT-4o, GPT-3.5-turbo |
| Anthropic | `anthropic/` | Claude Opus 4.5/4.6, Claude Sonnet 4.6, Claude Haiku 4.5 |
| Zhipu (智谱) | `anthropic/` | GLM-4, GLM-4-Flash, GLM-4.7 |
| DeepSeek | `openai_compat/` | DeepSeek Chat, DeepSeek Reasoner |
| Gemini | `openai_compat/` | Gemini Pro, Gemini Ultra |
| Groq | `openai_compat/` | Llama, Mixtral (fast inference) |
| Moonshot | `openai_compat/` | Moonshot-v1 |
| Qwen (通义千问) | `openai_compat/` | Qwen-Turbo, Qwen-Plus, Qwen-Max |
| NVIDIA NIM | `openai_compat/` | NVIDIA hosted models |
| Ollama | `openai_compat/` | Local models via Ollama |
| OpenRouter | `openai_compat/` | Multi-provider gateway |
| vLLM | `openai_compat/` | High-performance inference |
| Cerebras | `openai_compat/` | Fast inference |
| Volcengine (火山引擎) | `openai_compat/` | Doubao models |
| Claude CLI | `claude_cli_provider.go` | Claude via CLI |
| Codex CLI | `codex_cli_provider.go` | Codex via CLI |
| GitHub Copilot | `github_copilot_provider.go` | GitHub Copilot integration |

**Model Format**: `provider/model` (e.g., `openai/gpt-4`, `anthropic/claude-opus-4-6`, `zhipu/glm-4.7`)

---

## Available Tools (`pkg/tools/`)

| Tool | File | Description |
| --- | --- | --- |
| Cron | `cron.go` | Scheduled task management |
| Edit | `edit.go` | File editing operations |
| Filesystem | `filesystem.go` | File read/write operations |
| I2C | `i2c.go` | I2C hardware interface (Linux) |
| MCP Tool | `mcp_tool.go` | Model Context Protocol integration |
| Memory | `memory_tool.go` | Long-term memory operations |
| Message | `message.go` | Message handling utilities |
| Search | `search_tool.go` | Web search capabilities |
| Send File | `send_file.go` | File sending via channels |
| Shell | `shell.go` | Shell command execution |
| Shield Adapter | `shield_adapter.go` | SHIELD security integration |
| Skills Install | `skills_install.go` | Skill installation |
| Skills Search | `skills_search.go` | Skill discovery |
| Spawn | `spawn.go` | Process spawning |
| SPI | `spi.go` | SPI hardware interface (Linux) |
| Subagent | `subagent.go` | Sub-agent delegation |
| Web | `web.go` | HTTP/web operations |

---

## Workspace Structure (`~/.moonhub/workspace/`)

```
workspace/
├── sessions/          # Conversation sessions
├── memory/           # Long-term memory (MEMORY.md)
├── state/            # Persistent state
├── cron/             # Scheduled tasks database
├── skills/           # Custom skills
├── AGENTS.md         # Agent behavior instructions
├── HEARTBEAT.md      # Periodic task prompts
├── IDENTITY.md       # Agent identity settings
├── SOUL.md           # Agent personality
├── USER.md           # User preferences
└── SHIELD.md         # Security policies (optional)
```

---

## Quick Reference Links

| Resource | Link |
| --- | --- |
| Main README | [`../README.md`](../README.md) |
| CLAUDE.md (Dev Guide) | [`../CLAUDE.md`](../CLAUDE.md) |
| Config Example | [`../config/config.example.json`](../config/config.example.json) |
| Makefile | [`../Makefile`](../Makefile) |
| Docker Support | [`../docker/`](../docker/) |
