# Dynamic tools (`pkg/dynamictools`)

**Repository documentation index**: [docs/README.md](../../../docs/README.md)

## Overview

This package implements **Phase 1** of MoonHub **dynamic UI tools**: the backend stores **AI-generated tool definitions** (chat + space JSON schemas), deduplicates by content hash, and **executes** them with the **SchemaEngine** — fetching optional remote data through **HostFunctions**, then merging data into a copy of the UI schema for the PWA **DynamicRenderer**.

The **`engine`** field on `DynamicTool` allows `"schema"` (implemented) and `"wasm"` (reserved for a future runtime). Generation is triggered from the HTTP API via the configured LLM (see `web/backend/api/dynamic_tools.go`).

## Layout

| File | Role |
| --- | --- |
| `types.go` | `DynamicTool`, `GeneratedComponent`, `FetchConfig`, `GenerateRequest`, `ExecutionResult` |
| `tool_manager.go` | SQLite CRUD, `content_hash` unique index, `is_on_home` flag |
| `schema_engine.go` | `Execute` / `ExecuteSpace`, data injection into schema |
| `host_functions.go` | `HTTPFetch`, `ReadChannel` (stub); bounded response size |
| `fetch_url.go` | URL allowlisting / validation for fetches |

## Persistence

- Database path: **`<MOONHUB_HOME>/dynamic_tools.db`** (same home as `MOONHUB_HOME` / default `~/.moonhub`).
- Table `dynamic_tools` — see `tool_manager.go` for columns.

## Integration

- **HTTP**: [web/backend/api/dynamic_tools.go](../../../web/backend/api/dynamic_tools.go) registers LAN-only `/api/dynamic-tools` routes when initialization succeeds; if SQLite fails, routes are omitted (see `router.go` `initDynamicTools`).
- **Frontend**: MoonHub-PWA uses `DynamicToolsService` and `DynamicRenderer` + `components/chat/dynamic` / `components/space/dynamic`.
- **Status / checklist**: [docs/implementation/dynamic-tools-status.md](../../../docs/implementation/dynamic-tools-status.md)

## Tests

```bash
go test ./pkg/dynamictools/... -count=1
```
