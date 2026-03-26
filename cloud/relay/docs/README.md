# Cloud relay (WebSocket bridge)

In-process WebSocket relay that authenticates agents against the directory, then bridges pairs of connections after a simple text handshake.

**Repository documentation index**: [docs/README.md](../../../docs/README.md)

## Scope

- **Library**: `cloud/relay` — `Authenticator` (directory-backed Ed25519 verify), `Bridge` (agent registry + message forwarding).
- **Binary**: `cmd/relay` — HTTP server upgrading to WebSocket, challenge-based Bearer auth, delegates to `Bridge`.

## Authentication

1. Client sends a WebSocket upgrade request with `Authorization: Bearer <token>`.
2. Server generates a unique challenge (`GenerateChallenge`) and verifies the token:
   - Token format: `agentID:base64(signature)` where the signature is **Ed25519** over the **challenge bytes** (UTF-8 string).
3. Public key is loaded from the directory: `GET {directoryURL}/agents/{agentID}/pubkey`.
4. On success, `X-Agent-ID` is set internally and the connection is registered on the bridge.

## Wire protocol (text frames)

| Message | Meaning |
| --- | --- |
| `PING` | Server replies with `PONG`. |
| `CONNECT <targetAgentID>` | If target is connected, both sides receive `CONNECTED` and two goroutines forward binary/text frames between peers (control strings above are skipped on the forward path). |
| *(from server)* `ERROR target_offline` | Target not registered on this relay. |
| *(from server)* `ERROR peer_disconnected` | The other peer closed or failed. |

## Source map

| File | Role |
| --- | --- |
| `auth.go` | `Authenticator`, `Verify`, `GenerateChallenge` |
| `bridge.go` | `Bridge`, `ServeHTTP`, connect/forward logic |
| `*_test.go` | Unit tests |

## Configuration

See [CONFIG.md](CONFIG.md).

## Related documentation

- Directory HTTP API: [cloud/directory/docs/README.md](../../directory/docs/README.md)
- Device transport + resolver: [pkg/transport/docs/README.md](../../../pkg/transport/docs/README.md)
- Status: [docs/implementation/cloud-directory-relay-status.md](../../../docs/implementation/cloud-directory-relay-status.md)
