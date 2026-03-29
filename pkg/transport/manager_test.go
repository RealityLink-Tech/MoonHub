package transport

import (
	"crypto/ed25519"
	_ "net/http"
	_ "net/http/httptest"
	_ "strings"
	"testing"

	_ "github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
	_ "github.com/gorilla/websocket"
)

func TestManager_GetOrCreate(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)

	manager := NewManager("mh_local0000000000", priv)

	conn1, err := manager.GetOrCreate("mh_remote0000000000", "ws://localhost:1")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	conn2, err := manager.GetOrCreate("mh_remote0000000000", "ws://localhost:2")
	if err != nil {
		t.Fatalf("second GetOrCreate failed: %v", err)
	}

	if conn1 != conn2 {
		t.Error("expected same connection instance for same agent ID")
	}
}

func TestManager_Remove(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager := NewManager("mh_local0000000000", priv)

	conn, err := manager.GetOrCreate("mh_remote0000000000", "ws://localhost:1")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	manager.Remove("mh_remote0000000000")

	conn2, err := manager.GetOrCreate("mh_remote0000000000", "ws://localhost:1")
	if err != nil {
		t.Fatalf("GetOrCreate after remove failed: %v", err)
	}
	if conn2 == conn {
		t.Error("expected new connection instance after remove")
	}
}

func TestManager_List(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager := NewManager("mh_local0000000000", priv)

	_, _ = manager.GetOrCreate("mh_remote0000000000", "ws://localhost:1")
	_, _ = manager.GetOrCreate("mh_remote0000000001", "ws://localhost:2")

	ids := manager.List()
	if len(ids) != 2 {
		t.Errorf("expected 2 connections, got %d", len(ids))
	}
}

func TestManager_LocalAgentID(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager := NewManager("mh_local0000000000", priv)
	if manager.LocalAgentID() != "mh_local0000000000" {
		t.Errorf("expected mh_local0000000000, got %s", manager.LocalAgentID())
	}
}

func TestManager_Get_Nonexistent(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager := NewManager("mh_local0000000000", priv)

	conn := manager.Get("mh_nonexistent0000")
	if conn != nil {
		t.Error("expected nil for nonexistent connection")
	}
}

func TestManager_ResolveAndConnect_LAN(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	mgr := NewManager("mh_local0000000000", priv)

	cloud := &mockCloudClient{
		agents: map[string]*AgentLookupResult{
			"mh_remote1111111111": {
				Online:        true,
				RelayEndpoint: "wss://relay.example.com",
			},
		},
	}
	resolver := NewResolver(cloud)
	resolver.SetLANEndpoint("mh_remote1111111111", "ws://192.168.1.50:18801/agent/ws")
	mgr.SetResolver(resolver)

	conn, err := mgr.GetOrCreate("mh_remote1111111111", "")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.RemoteAgentID() != "mh_remote1111111111" {
		t.Errorf("remoteID = %q", conn.RemoteAgentID())
	}
}

func TestManager_ResolveAndConnect_Cloud(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	mgr := NewManager("mh_local0000000000", priv)

	cloud := &mockCloudClient{
		agents: map[string]*AgentLookupResult{
			"mh_remote1111111111": {
				Online:        true,
				RelayEndpoint: "wss://relay.example.com",
			},
		},
	}
	resolver := NewResolver(cloud)
	mgr.SetResolver(resolver)

	conn, err := mgr.GetOrCreate("mh_remote1111111111", "")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.RemoteAgentID() != "mh_remote1111111111" {
		t.Errorf("remoteID = %q", conn.RemoteAgentID())
	}
}

func TestManager_Resolve_Offline(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	mgr := NewManager("mh_local0000000000", priv)

	cloud := &mockCloudClient{agents: make(map[string]*AgentLookupResult)}
	resolver := NewResolver(cloud)
	mgr.SetResolver(resolver)

	_, err := mgr.GetOrCreate("mh_offline000000000", "")
	if err == nil {
		t.Error("expected error for offline agent")
	}
}
