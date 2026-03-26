# Transport (agent-to-agent)

Connection management and path selection between **LAN-direct WebSockets** and **cloud relay** URLs backed by the directory service.

**Repository documentation index**: [docs/README.md](../../../docs/README.md)

## Components

| Symbol | File | Role |
| --- | --- | --- |
| `Manager` | `manager.go` | Per-remote `AgentConn` pool; optional `Resolver` when `wsURL` is empty |
| `Resolver` | `resolver.go` | `Resolve(agentID)` — prefers LAN map, then cloud `LookupAgent` when online |
| `CloudClient` | `cloud.go` | HTTP client for directory: register, heartbeat, unregister, lookup, pubkey |
| `AgentConn`, `NewAgentConn` | `conn.go` | WebSocket client connection between agents |

## Cloud integration

- Implement `CloudLookup` (see `resolver.go`) with `*CloudClient` (`LookupAgent` matches the interface).
- `AgentLookupResult.RelayEndpoint` is returned as the WebSocket URL when resolving in cloud mode.
- Registration heartbeats: `CloudClient.StartHeartbeat(ctx, interval)` / `Stop()` (unregister).

Directory API details: [cloud/directory/docs/README.md](../../../cloud/directory/docs/README.md).

Relay handshake and Bearer token format: [cloud/relay/docs/README.md](../../../cloud/relay/docs/README.md).

## Configuration

There is no `config.json` block for transport yet; callers construct `NewCloudClient(directoryURL, identity)` and `NewResolver(cloud)` in code. See [cloud/directory/docs/CONFIG.md](../../../cloud/directory/docs/CONFIG.md) and [cloud/relay/docs/CONFIG.md](../../../cloud/relay/docs/CONFIG.md) for service-side settings.

## Tests

- `go test ./pkg/transport/...` — includes `cloud_test.go`, `integration_test.go` against the directory handler.

## Implementation status

[`docs/implementation/cloud-directory-relay-status.md`](../../../docs/implementation/cloud-directory-relay-status.md)
