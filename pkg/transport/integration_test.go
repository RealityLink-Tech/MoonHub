// pkg/transport/integration_test.go
package transport

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/cloud/directory"
	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
)

// testStore implements directory.AgentStore in memory.
type testStore struct {
	agents map[string]*directory.AgentRecord
}

func newTestStore() *testStore {
	return &testStore{agents: make(map[string]*directory.AgentRecord)}
}

func (s *testStore) UpsertAgent(ctx context.Context, agent *directory.AgentRecord) error {
	s.agents[agent.AgentID] = agent
	return nil
}

func (s *testStore) GetAgent(ctx context.Context, agentID string) (*directory.AgentRecord, error) {
	a, ok := s.agents[agentID]
	if !ok {
		return nil, directory.ErrAgentNotFound
	}
	return a, nil
}

func (s *testStore) DeleteAgent(ctx context.Context, agentID string) error {
	delete(s.agents, agentID)
	return nil
}

func (s *testStore) GetPublicKey(ctx context.Context, agentID string) ([]byte, error) {
	a, ok := s.agents[agentID]
	if !ok {
		return nil, directory.ErrAgentNotFound
	}
	return a.PublicKey, nil
}

// testCache implements directory.OnlineCache in memory.
type testCache struct {
	data map[string]string
}

func newTestCache() *testCache {
	return &testCache{data: make(map[string]string)}
}

func (c *testCache) MarkOnline(ctx context.Context, agentID, endpoint string) error {
	c.data[agentID] = endpoint
	return nil
}

func (c *testCache) IsOnline(ctx context.Context, agentID string) (bool, string, error) {
	endpoint, ok := c.data[agentID]
	return ok, endpoint, nil
}

func (c *testCache) MarkOffline(ctx context.Context, agentID string) error {
	delete(c.data, agentID)
	return nil
}

func TestIntegration_FullRegistrationAndLookup(t *testing.T) {
	// 1. Set up directory server with in-memory store/cache
	store := newTestStore()
	cache := newTestCache()
	handler := directory.NewHandler(store, cache)
	dirServer := httptest.NewServer(handler)
	defer dirServer.Close()

	// 2. Create two agent identities
	aliceIdentity, err := agentidentity.NewAgentIdentity("Alice")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}

	bobIdentity, err := agentidentity.NewAgentIdentity("Bob")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}

	// 3. Alice registers with cloud
	aliceClient := NewCloudClient(dirServer.URL, aliceIdentity)
	if err := aliceClient.Register("wss://relay.example.com"); err != nil {
		t.Fatalf("alice register: %v", err)
	}

	// 4. Bob registers with cloud
	bobClient := NewCloudClient(dirServer.URL, bobIdentity)
	if err := bobClient.Register("wss://relay.example.com"); err != nil {
		t.Fatalf("bob register: %v", err)
	}

	// 5. Alice looks up Bob via resolver
	resolver := NewResolver(aliceClient)
	strategy, err := resolver.Resolve(bobIdentity.AgentID)
	if err != nil {
		t.Fatalf("resolve bob: %v", err)
	}

	if strategy.Mode != ModeCloud {
		t.Errorf("expected ModeCloud, got %d", strategy.Mode)
	}
	if strategy.URL != "wss://relay.example.com" {
		t.Errorf("expected relay URL, got %q", strategy.URL)
	}

	// 6. Alice gets Bob's public key
	bobPub, err := aliceClient.GetPublicKey(bobIdentity.AgentID)
	if err != nil {
		t.Fatalf("get bob pubkey: %v", err)
	}
	if string(bobPub) != string(bobIdentity.PublicKeyBytes()) {
		t.Error("public key mismatch")
	}
}

func TestIntegration_LANPriorityOverCloud(t *testing.T) {
	// 1. Set up directory server
	store := newTestStore()
	cache := newTestCache()
	handler := directory.NewHandler(store, cache)
	dirServer := httptest.NewServer(handler)
	defer dirServer.Close()

	// 2. Create agent identities
	aliceIdentity, _ := agentidentity.NewAgentIdentity("Alice")
	bobIdentity, _ := agentidentity.NewAgentIdentity("Bob")

	// 3. Bob registers with cloud
	bobClient := NewCloudClient(dirServer.URL, bobIdentity)
	bobClient.Register("wss://relay.example.com")

	// 4. Alice creates resolver with both LAN and cloud
	aliceClient := NewCloudClient(dirServer.URL, aliceIdentity)
	resolver := NewResolver(aliceClient)

	// 5. Register Bob as LAN-discovered
	resolver.SetLANEndpoint(bobIdentity.AgentID, "ws://192.168.1.100:18801")

	// 6. Resolve should prefer LAN
	strategy, err := resolver.Resolve(bobIdentity.AgentID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if strategy.Mode != ModeLAN {
		t.Errorf("expected ModeLAN, got %d", strategy.Mode)
	}
	if strategy.URL != "ws://192.168.1.100:18801" {
		t.Errorf("expected LAN URL, got %q", strategy.URL)
	}
}
