// cloud/directory/store_test.go
package directory

import (
	"context"
	"testing"
)

// memoryStore is a test double implementing AgentStore in memory.
type memoryStore struct {
	agents map[string]*AgentRecord
}

func newMemoryStore() *memoryStore {
	return &memoryStore{agents: make(map[string]*AgentRecord)}
}

func (s *memoryStore) UpsertAgent(ctx context.Context, agent *AgentRecord) error {
	s.agents[agent.AgentID] = agent
	return nil
}

func (s *memoryStore) GetAgent(ctx context.Context, agentID string) (*AgentRecord, error) {
	a, ok := s.agents[agentID]
	if !ok {
		return nil, ErrAgentNotFound
	}
	return a, nil
}

func (s *memoryStore) DeleteAgent(ctx context.Context, agentID string) error {
	delete(s.agents, agentID)
	return nil
}

func (s *memoryStore) GetPublicKey(ctx context.Context, agentID string) ([]byte, error) {
	a, ok := s.agents[agentID]
	if !ok {
		return nil, ErrAgentNotFound
	}
	return a.PublicKey, nil
}

func TestMemoryStore_UpsertAndGet(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	agent := &AgentRecord{
		AgentID:    "mh_aaaa1111bbbb2222",
		AgentName:  "Test Agent",
		PublicKey:  []byte("test-pubkey"),
		Endpoint:   "wss://relay.example.com",
	}

	if err := store.UpsertAgent(ctx, agent); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetAgent(ctx, "mh_aaaa1111bbbb2222")
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentName != "Test Agent" {
		t.Errorf("name = %q, want %q", got.AgentName, "Test Agent")
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := newMemoryStore()
	_, err := store.GetAgent(context.Background(), "mh_nonexistent")
	if err != ErrAgentNotFound {
		t.Errorf("error = %v, want ErrAgentNotFound", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	store.UpsertAgent(ctx, &AgentRecord{
		AgentID:   "mh_aaaa1111bbbb2222",
		PublicKey: []byte("key"),
	})

	if err := store.DeleteAgent(ctx, "mh_aaaa1111bbbb2222"); err != nil {
		t.Fatal(err)
	}

	_, err := store.GetAgent(ctx, "mh_aaaa1111bbbb2222")
	if err != ErrAgentNotFound {
		t.Errorf("after delete, error = %v, want ErrAgentNotFound", err)
	}
}

func TestMemoryStore_GetPublicKey(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	store.UpsertAgent(ctx, &AgentRecord{
		AgentID:    "mh_aaaa1111bbbb2222",
		PublicKey:  []byte("my-public-key"),
		AgentName:  "Agent",
		Endpoint:   "wss://relay.example.com",
	})

	key, err := store.GetPublicKey(ctx, "mh_aaaa1111bbbb2222")
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != "my-public-key" {
		t.Errorf("key = %q, want %q", string(key), "my-public-key")
	}
}
