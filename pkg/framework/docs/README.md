# MoonHub Plugin Framework (`pkg/framework`)

This directory is the **plugin runtime core**: type definitions, global registry, lifecycle management. Implementation code is in the parent directory (`../*.go`), with Go package name `**plugin**` and import path `**github.com/RealityLink-Tech/MoonHub/pkg/framework**`.

## Recommended Reading Order

1. **This document** — Positioning and terminology
2. [ARCHITECTURE.md](./ARCHITECTURE.md) — Interfaces, Registry, lifecycle
3. [INTEGRATION.md](./INTEGRATION.md) — Integration with Gateway, channels, providers, agent

Implementation status and phase details: [`docs/implementation/plugin-status.md`](../../../docs/implementation/plugin-status.md).

## Package Name and Import (Common Confusion)

| What You See | Meaning |
|--------------|---------|
| Directory `pkg/framework/` | Physical path |
| `package plugin` | Go package name, so code uses `plugin.GlobalRegistry()`, `plugin.Metadata{}` |
| `import ".../pkg/framework"` | Standard library-style import, **does not** provide `framework.` prefix |

## What This Package Does NOT Do

- **Does not** enumerate business plugins in `init()`; specific Channel / Provider / Tool implementations are in `[pkg/plugins](../../plugins/docs/README.md)`, registered via blank imports by **Gateway**.
- `**LoadBuiltin()`** is a no-op retained for compatibility; see `builtin.go` comments.
