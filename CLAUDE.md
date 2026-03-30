# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MoonHub is an AI assistant that works out of the box — instant, alive, connected. Written in Go as a single binary for easy deployment. It's a multi-channel AI assistant framework supporting Telegram, Discord, Slack, Matrix, QQ, WeChat, and more, with dynamic UI generation and a global agent network.

## Build Commands

```bash
# Build for current platform
make build

# Build for all platforms (Linux, Darwin, Windows, NetBSD)
make build-all

# Build web interface launcher
make build-launcher

# Build for specific platforms
make build-linux-arm      # ARMv7 (Raspberry Pi Zero 2 W 32-bit)
make build-linux-arm64    # ARM64 (Raspberry Pi Zero 2 W 64-bit)
make build-pi-zero        # Both Pi Zero builds
```

## Development Commands

```bash
# Download dependencies
make deps

# Run code generation
make generate

# Run tests
make test

# Format code
make fmt

# Run linters
make lint

# Fix linting issues
make fix

# Run vet, fmt, verify deps, and test
make check
```

## Installation

```bash
# Install to ~/.local/bin
make install

# Uninstall binary only
make uninstall

# Uninstall binary and all data (~/.moonhub)
make uninstall-all
```

## Web Interface Development

```bash
cd web

# Run both frontend and backend dev servers
make dev

# Build frontend and embed into Go binary
make build

# Run backend tests and frontend lint
make test
```

## Docker Commands

```bash
# Build minimal Docker image
make docker-build

# Build full-featured Docker image (with Node.js 24 for MCP)
make docker-build-full

# Run gateway in Docker
make docker-run

# Test MCP tools in Docker
make docker-test
```

## Architecture

### Core Packages (pkg/)

- **agent/** - Main agent loop, message handling, tool execution
- **channels/** - Platform integrations (Telegram, Discord, Slack, Matrix, QQ, WeChat, etc.)
- **providers/** - LLM provider integrations (OpenAI, Anthropic, Gemini, Zhipu, etc.)
- **tools/** - Available tools (web search, file operations, cron, etc.)
- **skills/** - Extensible skill system for adding capabilities
- **config/** - Configuration management and loading
- **memory/** - Long-term memory system (MEMORY.md)
- **session/** - Session and conversation management
- **bus/** - Internal event bus for channel communication
- **commands/** - Slash command definitions and execution
- **routing/** - Message routing between channels and agents
- **mcp/** - Model Context Protocol integration
- **provisioning/** - Optional device WiFi provisioning, hotspot, recovery, auth code (used by web launcher when enabled)
- **transport/** - Agent-to-agent WebSocket paths: `Manager`, `Resolver` (LAN-first then cloud directory), `CloudClient` for directory HTTP; see [pkg/transport/docs/README.md](pkg/transport/docs/README.md)
- **dynamictools/** - AI-generated dynamic tools (SQLite `ToolManager`, `SchemaEngine`, LAN `/api/dynamic-tools`); see [pkg/dynamictools/docs/README.md](pkg/dynamictools/docs/README.md)

### Optional cloud services (`cloud/`, `cmd/`)

- **cloud/directory/** - HTTP agent directory (signed register/heartbeat, PostgreSQL, Redis online cache); binary: `cmd/directory-service` — [cloud/directory/docs/README.md](cloud/directory/docs/README.md)
- **cloud/relay/** - WebSocket relay with directory-backed Bearer+challenge auth; binary: `cmd/relay` — [cloud/relay/docs/README.md](cloud/relay/docs/README.md)

### CLI Commands (cmd/moonhub/internal/)

- **agent/** - Interactive chat mode (`moonhub agent`)
- **gateway/** - Long-running bot server for multi-channel support (`moonhub gateway`)
- **onboard/** - Initial setup wizard (`moonhub onboard`)
- **auth/** - OAuth authentication management (`moonhub auth`)
- **cron/** - Scheduled task management (`moonhub cron`)
- **skills/** - Skills management commands
- **status/** - System status display
- **version/** - Version information

### Web Interface (web/)

- **frontend/** - React + Vite + TanStack Router dashboard
- **backend/** - Go web server with embedded frontend; optional `/api/provisioning/*` and `/provisioning` UI when `MOONHUB_PROVISIONING_ENABLED=1`; optional `/api/dynamic-tools` when SQLite opens `<MOONHUB_HOME>/dynamic_tools.db` (see `pkg/dynamictools`)

### Key Entry Points

- `cmd/moonhub/main.go` - Main CLI entry point (Cobra-based)
- `pkg/agent/loop.go` - Core agent message processing loop
- `pkg/channels/` - Channel adapters for each platform
- `pkg/providers/` - LLM provider implementations

## Configuration

- Config file: `~/.moonhub/config.json`
- Workspace: `~/.moonhub/workspace/`
- Environment variables:
  - `MOONHUB_CONFIG` - Override config file path
  - `MOONHUB_HOME` - Override data root directory
  - `MOONHUB_PROVISIONING_ENABLED` - Set to `1` on the **web launcher** to enable device provisioning API and UI
  - `MOONHUB_PROVISIONING_TOKEN` - Optional shared secret for `/api/provisioning/*` (recommended with `-public`)
  - `MOONHUB_ALLOW_SYSTEM_CONTROL` - Set to `1` to allow provisioning-triggered system restart hooks

Docs: [pkg/provisioning/docs/README.md](pkg/provisioning/docs/README.md), [pkg/provisioning/docs/CONFIG.md](pkg/provisioning/docs/CONFIG.md).

## Workspace Structure

```
~/.moonhub/workspace/
├── sessions/          # Conversation sessions
├── memory/           # Long-term memory (MEMORY.md)
├── state/            # Persistent state
├── cron/             # Scheduled tasks database
├── skills/           # Custom skills
├── AGENTS.md         # Agent behavior instructions
├── HEARTBEAT.md      # Periodic task prompts
├── IDENTITY.md       # Agent identity settings
├── SOUL.md           # Agent personality
└── USER.md           # User preferences
```

## Model Configuration

MoonHub uses a model-centric configuration with `model_list` in config.json. Format: `provider/model` (e.g., `openai/gpt-4`, `anthropic/claude-opus-4-5`, `zhipu/glm-4.7`).

Supported providers: openai, anthropic, zhipu, deepseek, gemini, groq, moonshot, qwen, nvidia, ollama, openrouter, vllm, cerebras, volcengine, and more.

## Adding New Channels

1. Create a new package in `pkg/channels/<channel-name>/`
2. Implement the channel interface (see existing channels as reference)
3. Register in `pkg/channels/channels.go`
4. Add configuration support in `pkg/config/`

## Adding New Tools

1. Create tool definition in `pkg/tools/<tool-name>/`
2. Implement the tool interface with `Execute()` method
3. Register in `pkg/tools/registry.go`
4. Add to tool list in agent initialization

## Skills System

Skills are defined in `workspace/skills/<skill-name>/SKILL.md` with optional companion files. Built-in skills are in the `skills/` directory at repository root.

## Key Design Principles

- Lightweight by design: Target <10MB RAM usage, single binary, minimal deployment cost
- Single binary deployment across platforms
- No external runtime dependencies (pure Go)
- Workspace-based state management
- Multi-channel message routing via internal bus

## Commit Convention (Clean Commit)

Follow the [Clean Commit](https://github.com/wgtechlabs/clean-commit) standard.

### Format

```
<emoji> <type>: <description>
<emoji> <type> (<scope>): <description>    # with scope
<emoji> <type>!: <description>              # breaking change
```

### The 9 Types

| Emoji | Type | Usage |
|-------|------|-------|
| 📦 | `new` | Adding new features, files, or capabilities |
| 🔧 | `update` | Changing existing code, refactoring, improvements |
| 🗑️ | `remove` | Removing code, files, features, or dependencies |
| 🔒 | `security` | Security fixes, patches, vulnerability resolutions |
| ⚙️ | `setup` | Project configs, CI/CD, tooling, build systems |
| ☕ | `chore` | Maintenance tasks, dependency updates, housekeeping |
| 🧪 | `test` | Adding, updating, or fixing tests |
| 📖 | `docs` | Documentation changes and updates |
| 🚀 | `release` | Version releases and release preparation |

### Rules

- Use lowercase for type
- Use `!` immediately after type (no space) to signal a breaking change
- Use present tense ("add" not "added")
- No period at the end
- Keep description under 72 characters

### Examples

```
📦 new: add Telegram channel support
🔧 update (agent): improve message handling
🗑️ remove: deprecated legacy auth method
🔒 security: fix token validation vulnerability
⚙️ setup: add GitHub Actions CI workflow
☕ chore: update Go dependencies
🧪 test: add unit tests for memory package
📖 docs: update installation guide
🚀 release: version 1.0.0
```

## Don't add it to the commit：![1774768111676](image/CLAUDE/1774768111676.png)![1774768117875](image/CLAUDE/1774768117875.png)![1774768132477](image/CLAUDE/1774768132477.png)Co-Authored-By: Claude Opus 4.6 noreply@anthropic.com