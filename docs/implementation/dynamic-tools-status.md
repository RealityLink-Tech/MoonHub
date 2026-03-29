# Dynamic tools — implementation status

## Scope

Server-side **dynamic tools** for AI-generated UI: SQLite-backed definitions, LLM-assisted **generate**, **schema-driven execute** (chat vs space), and LAN HTTP API consumed by **MoonHub-PWA**.

## Shipped

- [x] `pkg/dynamictools` — `ToolManager`, `SchemaEngine`, `HostFunctions` (`HTTPFetch` with URL validation, response size cap), `types` aligned with PWA `GeneratedComponent`
- [x] `web/backend/api/dynamic_tools.go` — `GET/POST/PATCH/DELETE` under `/api/dynamic-tools` (LAN-only); generate uses device `config.json` LLM
- [x] `web/backend/api/router.go` — registers handler when DB init succeeds
- [x] MoonHub-PWA — `dynamicToolsService`, Space home / detail / add flows, `DynamicRenderer` + chat/space dynamic component sets

## API (summary)

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/dynamic-tools?source=ai` | List tools |
| POST | `/api/dynamic-tools/generate` | Create or return deduped tool from prompt |
| POST | `/api/dynamic-tools/{id}/execute` | Run tool (`mode`: `chat` \| `space`) |
| GET | `/api/dynamic-tools/{id}/schema` | Raw schema for a mode |
| DELETE | `/api/dynamic-tools/{id}` | Remove tool |
| PATCH | `/api/dynamic-tools/{id}/home` | Pin/unpin on Space home |

Authoritative tables: [web/backend/api/README.md](../../web/backend/api/README.md).

## Not yet / roadmap

- [ ] **`engine: wasm`** — no wazero (or other) executor wired; field reserved
- [ ] **ReadChannel** — host function stub returns empty
- [ ] Optional **config.json** keys dedicated to dynamic-tools (timeouts, model override) — currently reuses main LLM configuration from generate path

## Related docs

- [pkg/dynamictools/docs/README.md](../../pkg/dynamictools/docs/README.md)
- [MoonHub-PWA docs/services.md](../../MoonHub-PWA/docs/services.md) (client usage; split-repo layout)
