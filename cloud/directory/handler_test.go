// cloud/directory/handler_test.go
package directory

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
)

func testAgent(t *testing.T) (*agentidentity.AgentIdentity, []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := agentidentity.LoadFromKeys("TestAgent", priv, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return identity, pub
}

func signRegister(identity *agentidentity.AgentIdentity, agentID, agentName, pubKeyB64 string, timestamp int64) string {
	msg := fmt.Sprintf("%s%s%s%d", agentID, agentName, pubKeyB64, timestamp)
	sig := identity.Sign([]byte(msg))
	return base64.StdEncoding.EncodeToString(sig)
}

func TestHandler_RegisterAndGet(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: signRegister(identity, identity.AgentID, "TestAgent", pubKeyB64, timestamp),
	}

	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("register: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Now GET the agent
	req2 := httptest.NewRequest("GET", "/agents/"+identity.AgentID, nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("get: status = %d", rec2.Code)
	}

	var resp AgentResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.AgentID != identity.AgentID {
		t.Errorf("agentID = %q, want %q", resp.AgentID, identity.AgentID)
	}
	if resp.Online != true {
		t.Error("expected online=true")
	}
}

func TestHandler_Register_InvalidSignature(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString([]byte("invalid-sig")),
	}

	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandler_GetNotFound(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	req := httptest.NewRequest("GET", "/agents/mh_nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandler_Heartbeat(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	// Register first
	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: signRegister(identity, identity.AgentID, "TestAgent", pubKeyB64, timestamp),
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register failed: %d", rec.Code)
	}

	// Heartbeat - for heartbeat/delete, the signed message is agentID + timestamp
	heartbeatMsg := fmt.Sprintf("%s%d", identity.AgentID, timestamp)
	heartbeatSig := identity.Sign([]byte(heartbeatMsg))
	hbBody := HeartbeatRequest{
		Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString(heartbeatSig),
	}
	hbBytes, _ := json.Marshal(hbBody)
	req2 := httptest.NewRequest("PUT", "/agents/"+identity.AgentID+"/heartbeat", bytes.NewReader(hbBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("heartbeat: status = %d, body = %s", rec2.Code, rec2.Body.String())
	}
}

func TestHandler_Delete(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	// Register
	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: signRegister(identity, identity.AgentID, "TestAgent", pubKeyB64, timestamp),
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Delete - for heartbeat/delete, the signed message is agentID + timestamp
	deleteMsg := fmt.Sprintf("%s%d", identity.AgentID, timestamp)
	deleteSig := identity.Sign([]byte(deleteMsg))
	delBody := DeleteRequest{
		Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString(deleteSig),
	}
	delBytes, _ := json.Marshal(delBody)
	req2 := httptest.NewRequest("DELETE", "/agents/"+identity.AgentID, bytes.NewReader(delBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("delete: status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	// Verify deleted
	req3 := httptest.NewRequest("GET", "/agents/"+identity.AgentID, nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNotFound {
		t.Errorf("after delete, get status = %d, want 404", rec3.Code)
	}
}

func TestHandler_Register_PreventsKeyReplacement(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	// First registration with original key
	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: signRegister(identity, identity.AgentID, "TestAgent", pubKeyB64, timestamp),
	}

	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first register: %d", rec.Code)
	}

	// Try to register with a DIFFERENT key
	_, fakePriv, _ := ed25519.GenerateKey(nil)
	fakeIdentity, _ := agentidentity.LoadFromKeys("TestAgent", fakePriv, time.Now())

	fakePubB64 := base64.StdEncoding.EncodeToString(fakeIdentity.PublicKey)
	reRegBody := RegisterRequest{
		AgentID:   identity.AgentID, // Same agent ID
		AgentName: "TestAgent",
		PublicKey: fakePubB64, // Different public key
		Endpoint:  "wss://evil.example.com",
		Timestamp: time.Now().Unix(),
		// Sign with fake key — should fail because stored key doesn't match
		Signature: signRegister(fakeIdentity, identity.AgentID, "TestAgent", fakePubB64, time.Now().Unix()),
	}

	reBody, _ := json.Marshal(reRegBody)
	req2 := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(reBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("re-register with different key: status = %d, want 401", rec2.Code)
	}
}

func TestHandler_Heartbeat_ExpiredTimestamp(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	// Register first
	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: signRegister(identity, identity.AgentID, "TestAgent", pubKeyB64, timestamp),
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Heartbeat with expired timestamp (10 minutes ago)
	expiredTs := time.Now().Add(-10 * time.Minute).Unix()
	expiredMsg := fmt.Sprintf("%s%d", identity.AgentID, expiredTs)
	expiredSig := identity.Sign([]byte(expiredMsg))
	hbBody := HeartbeatRequest{
		Timestamp: expiredTs,
		Signature: base64.StdEncoding.EncodeToString(expiredSig),
	}
	hbBytes, _ := json.Marshal(hbBody)
	req2 := httptest.NewRequest("PUT", "/agents/"+identity.AgentID+"/heartbeat", bytes.NewReader(hbBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Errorf("expired heartbeat: status = %d, want 400", rec2.Code)
	}
}

func TestHandler_Delete_ExpiredTimestamp(t *testing.T) {
	store := newMemoryStore()
	cache := newMemoryCache()
	handler := NewHandler(store, cache)

	identity, pub := testAgent(t)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)
	timestamp := time.Now().Unix()

	// Register
	regBody := RegisterRequest{
		AgentID:   identity.AgentID,
		AgentName: "TestAgent",
		PublicKey: pubKeyB64,
		Endpoint:  "wss://relay.example.com",
		Timestamp: timestamp,
		Signature: signRegister(identity, identity.AgentID, "TestAgent", pubKeyB64, timestamp),
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Delete with expired timestamp
	expiredTs := time.Now().Add(-10 * time.Minute).Unix()
	expiredMsg := fmt.Sprintf("%s%d", identity.AgentID, expiredTs)
	expiredSig := identity.Sign([]byte(expiredMsg))
	delBody := DeleteRequest{
		Timestamp: expiredTs,
		Signature: base64.StdEncoding.EncodeToString(expiredSig),
	}
	delBytes, _ := json.Marshal(delBody)
	req2 := httptest.NewRequest("DELETE", "/agents/"+identity.AgentID, bytes.NewReader(delBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Errorf("expired delete: status = %d, want 400", rec2.Code)
	}
}
