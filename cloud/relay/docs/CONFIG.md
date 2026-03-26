# Cloud relay — configuration

## `cmd/relay` flags

| Flag | Default | Description |
| --- | --- | --- |
| `-addr` | `:8081` | HTTP/WebSocket listen address |
| `-directory-url` | `http://localhost:8080` | Base URL of the directory service (used to fetch agent public keys) |

No additional environment variables are required by the relay binary itself; ensure the directory URL is reachable from the relay host.
