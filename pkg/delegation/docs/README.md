# MoonHub Delegation (`pkg/delegation`)

Sub-agent orchestration: delegate tasks (sync, batch, background), role templates with reuse, adaptive timeouts, SQLite persistence, Intercom pub/sub, and session-scoped queues. **Off by default** until `delegation.enabled` is set in `config.json`.

**Repository documentation index** (subsystem reading order): [docs/README.md](../../../docs/README.md).

## Recommended Reading Order (Documentation Flow)

1. **This document** — Responsibility boundaries and source map
2. [CONFIG.md](./CONFIG.md) — `delegation` in `config.json`, environment variables, defaults
3. Implementation details — [`docs/implementation/delegation-status.md`](../../../docs/implementation/delegation-status.md) (schema, eight tools, Intercom topics, examples, tests)

## Source Map (Aligned with Implementation)

```
pkg/delegation/
├── types.go              # Core domain types
├── store.go              # SQLite persistence (sub agents, templates, tasks, metrics, messages)
├── intercom.go           # Pub/sub topics
├── timeout_estimator.go  # Category-based adaptive timeouts
├── templates.go          # Role template find/create and reuse (Jaccard / threshold)
├── lifecycle.go          # Sub-agent lifecycle
├── background.go         # Background task runner
├── queue.go              # Per-user session queue (serial / parallel scheduling)
├── tools.go              # DelegationSystem facade, tool registration
├── tools_delegate.go     # delegate_task, delegate_tasks
├── tools_background.go   # delegate_background, delegate_to_existing
├── tools_manage.go       # list_sub_agents, manage_sub_agent, manage_template, confirm_task
├── tools_helper.go       # Shared helpers
└── *_test.go
```

Agent wiring lives outside this package:

- `pkg/agent/delegation_integration.go` — construct system, register tools, bridge config
- `pkg/agent/instance.go` — enable when `cfg.Delegation.Enabled`
- `pkg/agent/loop.go` — inject background results; `ExecuteWithContext` + `DelegationUserID`

Testing: `go test ./pkg/delegation/... -v` (see implementation status for coverage / benches).

## Related Code (Integration Points)

| Area | Path | Description |
|------|------|-------------|
| Global config type | `pkg/config/config.go` (`DelegationConfig`) | User-visible `delegation` JSON and env tags |
| Defaults | `pkg/config/defaults.go` | `DefaultDelegationConfig()` |
| Tool context | `pkg/tools/…` | `DelegationUserID(ctx)` for partitioned store access |
| Agent integration | `pkg/agent/delegation_integration.go` | `NewDelegationIntegration`, `RegisterTools` |

## Alignment with Other Subsystems

This directory follows the same convention as [`pkg/learning/docs`](../../learning/docs/README.md), [`pkg/compactor/docs`](../../compactor/docs/README.md), and [`pkg/shield/docs`](../../shield/docs/README.md): **package-level `docs/`** covers configuration and where to read code; **repository-level `docs/implementation/delegation-status.md`** holds deep design, SQL schema, tool tables, and integration notes.
