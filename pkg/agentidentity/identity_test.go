package agentidentity

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestDeriveAgentID(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	id := DeriveAgentID(priv.Public().(ed25519.PublicKey))
	if len(id) != 19 { // "mh_" + 16 hex chars
		t.Errorf("expected length 19, got %d: %s", len(id), id)
	}
	if id[:3] != "mh_" {
		t.Errorf("expected prefix mh_, got %s", id[:3])
	}
}

func TestDeriveAgentID_Deterministic(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)

	id1 := DeriveAgentID(pub)
	id2 := DeriveAgentID(pub)

	if id1 != id2 {
		t.Errorf("same key should produce same ID: %s vs %s", id1, id2)
	}
}

func TestDeriveAgentID_DifferentKeys(t *testing.T) {
	pub1, _, _ := ed25519.GenerateKey(rand.Reader)
	pub2, _, _ := ed25519.GenerateKey(rand.Reader)

	id1 := DeriveAgentID(pub1)
	id2 := DeriveAgentID(pub2)

	if id1 == id2 {
		t.Error("different keys should produce different IDs")
	}
}

func TestNewAgentIdentity(t *testing.T) {
	ai, err := NewAgentIdentity("TestAgent")
	if err != nil {
		t.Fatal(err)
	}

	if ai.AgentName != "TestAgent" {
		t.Errorf("expected name TestAgent, got %s", ai.AgentName)
	}

	if ai.AgentID == "" {
		t.Error("expected non-empty AgentID")
	}

	if ai.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	if len(ai.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("expected public key size %d, got %d", ed25519.PublicKeySize, len(ai.PublicKey))
	}
}

func TestAgentIdentity_CertificateRoundtrip(t *testing.T) {
	ai, err := NewAgentIdentity("RoundtripTest")
	if err != nil {
		t.Fatal(err)
	}

	cert := ai.Certificate()
	if cert.AgentID != ai.AgentID {
		t.Errorf("cert AgentID mismatch: %s vs %s", cert.AgentID, ai.AgentID)
	}
	if cert.AgentName != ai.AgentName {
		t.Errorf("cert AgentName mismatch: %s vs %s", cert.AgentName, ai.AgentName)
	}
	if cert.CreatedAt.IsZero() {
		t.Error("expected non-zero cert CreatedAt")
	}
}

func TestAgentIdentity_SignAndVerify(t *testing.T) {
	ai, err := NewAgentIdentity("SignTest")
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("hello world")
	sig := ai.Sign(message)

	if !ai.Verify(message, sig) {
		t.Error("signature should verify")
	}

	// Tampered message
	if ai.Verify([]byte("tampered"), sig) {
		t.Error("tampered message should not verify")
	}
}

func TestAgentIdentity_PublicKeyBytes(t *testing.T) {
	ai, err := NewAgentIdentity("PubKeyTest")
	if err != nil {
		t.Fatal(err)
	}

	pubBytes := ai.PublicKeyBytes()
	if len(pubBytes) != ed25519.PublicKeySize {
		t.Errorf("expected %d bytes, got %d", ed25519.PublicKeySize, len(pubBytes))
	}
}

func TestAgentIdentity_LoadFromKeys(t *testing.T) {
	original, err := NewAgentIdentity("Original")
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFromKeys(original.AgentName, original.PrivateKey, original.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.AgentID != original.AgentID {
		t.Errorf("AgentID mismatch: %s vs %s", loaded.AgentID, original.AgentID)
	}
	if loaded.AgentName != original.AgentName {
		t.Errorf("AgentName mismatch: %s vs %s", loaded.AgentName, original.AgentName)
	}
}
