# MoonHub Web

This directory contains the standalone web service for `moonhub`.
It provides a complete unified web interface, acting as a dashboard, configuration center, and interactive console (channel client) for the core `moonhub` engine.

## Architecture

The service is structured as a monorepo containing both the backend and frontend code to ensure high cohesion and simplify deployment.

*   **`backend/`**: The Go-based web server. It provides RESTful APIs, manages WebSocket connections for chat, and handles the lifecycle of the `moonhub` process. It eventually embeds the compiled frontend assets into a single executable.
*   **`frontend/`**: The Vite + React + TanStack Router single-page application (SPA). It provides the interactive user interface.

### Device provisioning (optional)

Edge-style **WiFi provisioning** (hotspot, scan/connect, recovery, auth code, factory reset) is implemented in `pkg/provisioning/` and exposed by the launcher when `MOONHUB_PROVISIONING_ENABLED=1`.

| Area | Location |
| --- | --- |
| Go API + SSE | `web/backend/api/provisioning.go`, `web/backend/middleware/provisioning_auth.go` |
| Wiring + env | `web/backend/main.go` (`SetProvisioningHandler` **before** `RegisterRoutes`) |
| React UI | `web/frontend/src/routes/provisioning/`, `web/frontend/src/components/provisioning/` |
| PWA / offline (provisioning scope) | `web/frontend/public/sw.js`, `site.webmanifest` |

**Documentation flow** (same pattern as other subsystems in the repo): [pkg/provisioning/docs/README.md](../pkg/provisioning/docs/README.md) → [CONFIG.md](../pkg/provisioning/docs/CONFIG.md) → [docs/implementation/provisioning-status.md](../docs/implementation/provisioning-status.md).

### Companion PWA & LAN HTTP API

The installable **MoonHub-PWA** client (separate repository; often checked out beside this repo as `MoonHub-PWA/`) talks to the device over **LAN** using the Go API under [`backend/api/`](backend/api/). Operator-facing endpoint tables (mDNS `GET /api/discover`, paired list `GET /api/devices`, channel CRUD, auth, config, chat, gateway) live in [`backend/api/README.md`](backend/api/README.md).

## Getting Started

### Prerequisites

*   Go 1.25+
*   Node.js 20+ with pnpm

### Development

Run both the frontend dev server and the Go backend simultaneously:

```bash
make dev
```

Or run them separately:

```bash
make dev-frontend   # Vite dev server
make dev-backend    # Go backend
```

### Build

Build the frontend and embed it into a single Go binary:

```bash
make build
```

The output binary is `backend/moonhub-web`.

### Other Commands

```bash
make test    # Run backend tests and frontend lint
make lint    # Run go vet and prettier/eslint
make clean   # Remove all build artifacts
```
