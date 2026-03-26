// pkg/transport/resolver_test.go
package transport

import (
	"fmt"
	"testing"
)

// mockCloudClient implements CloudLookup for testing.
type mockCloudClient struct {
	agents map[string]*AgentLookupResult
}

func (m *mockCloudClient) LookupAgent(agentID string) (*AgentLookupResult, error) {
	a, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return a, nil
}

func TestResolver_LANAvailable(t *testing.T) {
	resolver := NewResolver(nil)
	resolver.SetLANEndpoint("mh_aaaa1111bbbb2222", "ws://192.168.1.100:18801/agent/ws")

	strategy, err := resolver.Resolve("mh_aaaa1111bbbb2222")
	if err != nil {
		t.Fatal(err)
	}
	if strategy.Mode != ModeLAN {
		t.Errorf("mode = %d, want ModeLAN", strategy.Mode)
	}
	if strategy.URL != "ws://192.168.1.100:18801/agent/ws" {
		t.Errorf("url = %q", strategy.URL)
	}
}

func TestResolver_CloudFallback(t *testing.T) {
	cloud := &mockCloudClient{
		agents: map[string]*AgentLookupResult{
			"mh_cccc3333dddd4444": {
				AgentID:       "mh_cccc3333dddd4444",
				Online:        true,
				RelayEndpoint: "wss://relay.example.com",
			},
		},
	}
	resolver := NewResolver(cloud)

	strategy, err := resolver.Resolve("mh_cccc3333dddd4444")
	if err != nil {
		t.Fatal(err)
	}
	if strategy.Mode != ModeCloud {
		t.Errorf("mode = %d, want ModeCloud", strategy.Mode)
	}
	if strategy.URL != "wss://relay.example.com" {
		t.Errorf("url = %q", strategy.URL)
	}
}

func TestResolver_Offline(t *testing.T) {
	cloud := &mockCloudClient{agents: make(map[string]*AgentLookupResult)}
	resolver := NewResolver(cloud)

	_, err := resolver.Resolve("mh_nonexistent")
	if err == nil {
		t.Error("expected error for offline agent")
	}
}

func TestResolver_LANPriority(t *testing.T) {
	cloud := &mockCloudClient{
		agents: map[string]*AgentLookupResult{
			"mh_aaaa1111bbbb2222": {
				Online:        true,
				RelayEndpoint: "wss://relay.example.com",
			},
		},
	}
	resolver := NewResolver(cloud)
	resolver.SetLANEndpoint("mh_aaaa1111bbbb2222", "ws://192.168.1.100:18801/agent/ws")

	strategy, err := resolver.Resolve("mh_aaaa1111bbbb2222")
	if err != nil {
		t.Fatal(err)
	}
	if strategy.Mode != ModeLAN {
		t.Errorf("mode = %d, want ModeLAN (priority)", strategy.Mode)
	}
}
