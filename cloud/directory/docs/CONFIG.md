# Cloud directory — configuration

## `cmd/directory-service` flags

| Flag | Default | Description |
| --- | --- | --- |
| `-addr` | `:8080` | HTTP listen address |
| `-db-host` | `localhost` | PostgreSQL host |
| `-db-port` | `5432` | PostgreSQL port |
| `-db-user` | `moonhub` | PostgreSQL user |
| `-db-pass` | *(empty)* | PostgreSQL password |
| `-db-name` | `moonhub` | Database name |

## Environment variables

When flags are still at their defaults, database settings can be overridden (useful for Docker):

| Variable | Overrides default for |
| --- | --- |
| `DB_HOST` | `-db-host` when it was `localhost` |
| `DB_PORT` | `-db-port` when it was `5432` |
| `DB_USER` | `-db-user` when it was `moonhub` |
| `DB_PASS` | `-db-pass` when empty |
| `DB_NAME` | `-db-name` when it was `moonhub` |

**Redis**

| Variable | Description |
| --- | --- |
| `REDIS_URL` | Redis URL for `NewRedisCache` (e.g. `redis://localhost:6379/0`). If unset or invalid, the client falls back to `localhost:6379`. |

On startup the process runs `directory.MigrationSQL` against PostgreSQL (creates `agents` table and index).
