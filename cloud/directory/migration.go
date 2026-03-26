// cloud/directory/migration.go
package directory

// MigrationSQL creates the agents table.
const MigrationSQL = `
CREATE TABLE IF NOT EXISTS agents (
    agent_id   TEXT PRIMARY KEY,
    agent_name TEXT NOT NULL,
    public_key BYTEA NOT NULL,
    endpoint   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agents_last_seen ON agents(last_seen_at);
`
