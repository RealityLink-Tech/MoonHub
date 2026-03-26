// pkg/transport/cloud_test.go
package transport

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
)

func TestCloudClient_Register(t *testing.T) {
	var receivedBody RegisterRequest
	dirServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agents/register" || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer dirServer.Close()

	_, priv, _ := ed25519.GenerateKey(nil)
	identity, _ := agentidentity.LoadFromKeys("TestAgent", priv, time.Now())

	client := NewCloudClient(dirServer.URL, identity)
	if err := client.Register("wss://relay.example.com"); err != nil {
		t.Fatal(err)
	}

	if receivedBody.AgentID != identity.AgentID {
		t.Errorf("agentID = %q, want %q", receivedBody.AgentID, identity.AgentID)
	}
	if receivedBody.Endpoint != "wss://relay.example.com" {
		t.Errorf("endpoint = %q", receivedBody.Endpoint)
	}
}

func TestCloudClient_LookupAgent(t *testing.T) {
	dirServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pubkey") {
			agentID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/agents/"), "/pubkey")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"agentID":   agentID,
				"publicKey": base64.StdEncoding.EncodeToString([]byte("test-key")),
			})
			return
		}
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/agents/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"agentID":              "mh_cccc3333dddd4444",
				"agentName":           "RemoteAgent",
				"online":              true,
				"publicKeyFingerprint": "abcd1234",
				"relayEndpoint":       "wss://relay.example.com",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer dirServer.Close()

	_, priv, _ := ed25519.GenerateKey(nil)
	identity, _ := agentidentity.LoadFromKeys("TestAgent", priv, time.Now())

	client := NewCloudClient(dirServer.URL, identity)
	result, err := client.LookupAgent("mh_cccc3333dddd4444")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Online {
		t.Error("expected online")
	}
	if result.RelayEndpoint != "wss://relay.example.com" {
		t.Errorf("relayEndpoint = %q", result.RelayEndpoint)
	}
}

func TestCloudClient_LookupAgent_Offline(t *testing.T) {
	dirServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/agents/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer dirServer.Close()

	_, priv, _ := ed25519.GenerateKey(nil)
	identity, _ := agentidentity.LoadFromKeys("TestAgent", priv, time.Now())

	client := NewCloudClient(dirServer.URL, identity)
	_, err := client.LookupAgent("mh_nonexistent")
	if err == nil {
		t.Error("expected error for offline agent")
	}
}

func TestCloudClient_GetPublicKey(t *testing.T) {
	dirServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pubkey") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"agentID":   "mh_cccc3333dddd4444",
				"publicKey": base64.StdEncoding.EncodeToString([]byte("remote-pub-key")),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer dirServer.Close()

	_, priv, _ := ed25519.GenerateKey(nil)
	identity, _ := agentidentity.LoadFromKeys("TestAgent", priv, time.Now())

	client := NewCloudClient(dirServer.URL, identity)
	key, err := client.GetPublicKey("mh_cccc3333dddd4444")
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != "remote-pub-key" {
		t.Errorf("key = %q, want %q", string(key), "remote-pub-key")
	}
}
