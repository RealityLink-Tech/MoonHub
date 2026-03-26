// cloud/directory/store.go
package directory

import (
	"context"
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
