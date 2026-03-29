// cloud/directory/store.go
package directory

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrAgentNotFound = errors.New("agent not found")

// AgentRecord represents a registered agent in the directory.
type AgentRecord struct {
	AgentID    string    `json:"agent_id" db:"agent_id"`
	AgentName  string    `json:"agent_name" db:"agent_name"`
	PublicKey  []byte    `json:"public_key" db:"public_key"`
	Endpoint   string    `json:"endpoint" db:"endpoint"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at" db:"last_seen_at"`
}

// AgentStore is the persistence interface for agent records.
type AgentStore interface {
	UpsertAgent(ctx context.Context, agent *AgentRecord) error
	GetAgent(ctx context.Context, agentID string) (*AgentRecord, error)
	DeleteAgent(ctx context.Context, agentID string) error
	GetPublicKey(ctx context.Context, agentID string) ([]byte, error)
}

// NewPostgresStore creates a PostgreSQL-backed AgentStore.
func NewPostgresStore(db *sql.DB) AgentStore {
	return &postgresStore{db: db}
}

type postgresStore struct {
	db *sql.DB
}

func (s *postgresStore) UpsertAgent(ctx context.Context, agent *AgentRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (agent_id, agent_name, public_key, endpoint, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (agent_id) DO UPDATE SET
			agent_name = EXCLUDED.agent_name,
			endpoint = EXCLUDED.endpoint,
			last_seen_at = EXCLUDED.last_seen_at`,
		agent.AgentID, agent.AgentName, agent.PublicKey, agent.Endpoint,
		agent.CreatedAt, agent.LastSeenAt)
	return err
}

func (s *postgresStore) GetAgent(ctx context.Context, agentID string) (*AgentRecord, error) {
	var a AgentRecord
	err := s.db.QueryRowContext(ctx,
		"SELECT agent_id, agent_name, public_key, endpoint, created_at, last_seen_at FROM agents WHERE agent_id = $1",
		agentID).Scan(&a.AgentID, &a.AgentName, &a.PublicKey, &a.Endpoint, &a.CreatedAt, &a.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *postgresStore) DeleteAgent(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM agents WHERE agent_id = $1", agentID)
	return err
}

func (s *postgresStore) GetPublicKey(ctx context.Context, agentID string) ([]byte, error) {
	var pubKey []byte
	err := s.db.QueryRowContext(ctx, "SELECT public_key FROM agents WHERE agent_id = $1", agentID).Scan(&pubKey)
	if err == sql.ErrNoRows {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}
