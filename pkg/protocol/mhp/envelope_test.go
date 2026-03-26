package mhp

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testKeypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func TestNewEnvelope(t *testing.T) {
	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgFriendRequest, nil)
	if e.Version != 1 {
		t.Errorf("expected version 1, got %d", e.Version)
	}
	if e.From != "mh_aaaa" {
		t.Errorf("expected mh_aaaa, got %s", e.From)
	}
	if e.Type != MsgFriendRequest {
		t.Errorf("expected friend_request, got %s", e.Type)
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
}

func TestEnvelope_Sign(t *testing.T) {
	_, priv := testKeypair(t)

	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgTaskRequest, TaskRequestPayload{
		Task: "Do something",
	})

	sig, err := e.Sign(priv)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) == 0 {
		t.Error("expected non-empty signature")
	}
	if e.Signature == "" {
		t.Error("signature field should be set after Sign")
	}
}

func TestEnvelope_Verify(t *testing.T) {
	pub, priv := testKeypair(t)

	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgTaskRequest, TaskRequestPayload{
		Task: "Do something",
		Zone: "shared",
	})

	if _, err := e.Sign(priv); err != nil {
		t.Fatal(err)
	}

	if err := e.Verify(pub); err != nil {
		t.Errorf("signature should verify: %v", err)
	}
}

func TestEnvelope_Verify_WrongKey(t *testing.T) {
	pub1, priv1 := testKeypair(t)
	_, _ = testKeypair(t)

	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgTaskRequest, nil)
	e.Sign(priv1)

	tampered := *e
	tampered.Payload = json.RawMessage(`{"evil": "payload"}`)
	if err := tampered.Verify(pub1); err == nil {
		t.Error("tampered payload should fail verification")
	}
}

func TestEnvelope_Verify_NoSignature(t *testing.T) {
	pub, _ := testKeypair(t)

	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgTaskRequest, nil)

	if err := e.Verify(pub); err == nil {
		t.Error("unsigned envelope should fail verification")
	}
}

func TestEnvelope_MarshalJSON(t *testing.T) {
	_, priv := testKeypair(t)

	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgFriendRequest, FriendRequestPayload{
		AgentName: "TestAgent",
	})
	e.Sign(priv)

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.From != "mh_aaaa" {
		t.Errorf("expected mh_aaaa, got %s", decoded.From)
	}
	if decoded.Type != MsgFriendRequest {
		t.Errorf("expected friend_request, got %s", decoded.Type)
	}
	if decoded.Signature == "" {
		t.Error("expected non-empty signature in decoded envelope")
	}
}

func TestEnvelope_Verify_ReplayDetection(t *testing.T) {
	_, priv := testKeypair(t)

	e := NewEnvelope("mh_aaaa", "mh_bbbb", MsgTaskRequest, nil)
	e.Sign(priv)

	dedup := NewMessageDedup()

	if !dedup.CheckAndAdd(e.ID, e.Timestamp) {
		t.Error("first message should pass dedup")
	}

	if dedup.CheckAndAdd(e.ID, e.Timestamp) {
		t.Error("duplicate message should fail dedup")
	}
}

func TestMessageDedup_StaleMessage(t *testing.T) {
	dedup := NewMessageDedup()

	staleTS := time.Now().Add(-6 * time.Minute).Unix()
	if dedup.CheckAndAdd("stale-msg-id", staleTS) {
		t.Error("stale message should fail dedup")
	}
}

func TestEnvelope_SignedFields(t *testing.T) {
	e := Envelope{
		Version:   1,
		From:      "mh_aaaa",
		To:        "mh_bbbb",
		Type:      MsgTaskRequest,
		ID:        "test-id",
		Timestamp: 1700000000,
		Payload:   json.RawMessage(`{"key":"value"}`),
	}

	signed := e.signedFields()
	expected := `1mh_aaaamh_bbbbtask_requesttest-id1700000000{"key":"value"}`

	if signed != expected {
		t.Errorf("signed fields mismatch:\ngot:      %s\nexpected: %s", signed, expected)
	}
}

func TestEnvelope_SignedFields_NilPayload(t *testing.T) {
	e := Envelope{
		Version:   1,
		From:      "mh_aaaa",
		To:        "mh_bbbb",
		Type:      MsgDiscoveryProbe,
		ID:        "test-id",
		Timestamp: 1700000000,
	}

	signed := e.signedFields()
	if strings.Contains(signed, "null") {
		t.Error("nil payload should produce empty object, not null")
	}
}
