# Reference templates (MoonHub doc sync)

Use these as structural guides; adapt paths and names to the actual change.

## CHANGELOG.md section skeleton

```markdown
## YYYY-MM-DD — Short Title

### Summary

One short paragraph: what shipped and why it matters.

### New Features

**Feature Area** (`pkg/example/`)
- Bullet capability
- Another capability

### Files Changed

- `pkg/example/` — What changed

### Documentation

- [`pkg/example/docs/README.md`](pkg/example/docs/README.md) — …
- [`docs/README.md`](docs/README.md) — Index updates
```

## README.md Implemented bullet pattern

```markdown
- **Title** — One line. See [`docs/implementation/foo-status.md`](docs/implementation/foo-status.md) and [`pkg/foo/docs/README.md`](pkg/foo/docs/README.md).
```

## README_CN.md 已实现 bullet pattern

```markdown
- **标题** — 一行说明。参见 [`docs/implementation/foo-status.md`](docs/implementation/foo-status.md) 与 [`pkg/foo/docs/README.md`](pkg/foo/docs/README.md)。
```

## docs/README.md subsystem table row pattern

```markdown
| 1 | [`pkg/foo/docs/README.md`](../pkg/foo/docs/README.md) | Scope, source map, integration |
| 2 | [`pkg/foo/docs/CONFIG.md`](../pkg/foo/docs/CONFIG.md) | `foo` configuration |
| Status | [`docs/implementation/foo-status.md`](./implementation/foo-status.md) | Implementation status |
```

## pkg/…/docs/README.md header pattern

- Title: subsystem name + short purpose.
- Link to index: `**Repository Documentation Index**: [docs/README.md](../../../docs/README.md)` (adjust `../` count from file depth).
- Sections: Overview, core components or source map, Configuration (link to CONFIG.md), Integration, optional Examples.
