# Cloud directory service

HTTP API that registers MoonHub agents (Ed25519 identity), tracks last-seen metadata in PostgreSQL, and exposes online status plus relay WebSocket endpoints via Redis.

**Repository documentation index**: [docs/README.md](../../../docs/README.md)

## Scope

- **Library**: `cloud/directory` — `Handler` (`http.Handler`), `AgentStore` (PostgreSQL), `OnlineCache` (Redis), schema migration SQL.
- **Binary**: `cmd/directory-service` — listens on HTTP, runs migrations, wires store + cache.

## HTTP API (summary)

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/agents/register` | Register or update agent; signed body; `agentID` must match derived ID from public key |
| `PUT` | `/agents/{id}/heartbeat` | Signed heartbeat; refreshes `last_seen` and online cache |
| `GET` | `/agents/{id}` | Lookup: name, online flag, public key fingerprint, `relayEndpoint` when online |
| `GET` | `/agents/{id}/pubkey` | Base64 public key (used by relay auth) |
| `DELETE` | `/agents/{id}` | Signed unregister; removes DB row and cache |

All mutating requests use Ed25519 signatures over canonical message strings with a **5-minute** timestamp window. Re-registration verifies signatures with the **stored** public key when the agent already exists (prevents key replacement).

## Source map

| File | Role |
| --- | --- |
| `handler.go` | Route dispatch, signature verification, JSON responses |
| `store.go` | `AgentStore` + `NewPostgresStore` |
| `cache.go` | `OnlineCache` + `NewRedisCache` |
| `migration.go` | `MigrationSQL` for `agents` table |
| `handler_test.go` | Handler tests |

## Configuration

See [CONFIG.md](CONFIG.md) for flags, environment variables, and Redis URL behavior.

## Integration

- **Device side**: [`pkg/transport/cloud.go`](../../../pkg/transport/cloud.go) — `CloudClient` calls the same paths for register, heartbeat, lookup, and pubkey fetch.
- **Relay**: [`cloud/relay/docs/README.md`](../../relay/docs/README.md) — authenticates WebSocket clients using `GET /agents/{id}/pubkey`.
- **Deploy**: [`deploy/docker-compose.yml`](../../../deploy/docker-compose.yml) can build the directory image from `cloud/directory/Dockerfile`.

## Implementation status

Deep checklist and Phase context: [`docs/implementation/cloud-directory-relay-status.md`](../../../docs/implementation/cloud-directory-relay-status.md).
