// Package agentidentity provides cryptographic agent identity for MoonHub.
// Each agent has a stable Ed25519 keypair and a derived AgentID.
package agentidentity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// AgentIDPrefix is the prefix for all agent IDs.
const AgentIDPrefix = "mh_"

// AgentCertificate is a self-signed certificate containing agent identity info.
type AgentCertificate struct {
	AgentID   string    `json:"agentId"`
	AgentName string    `json:"agentName"`
	CreatedAt time.Time `json:"createdAt"`
}

// AgentIdentity represents a MoonHub agent's cryptographic identity.
type AgentIdentity struct {
	AgentID    string
	AgentName  string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	CreatedAt  time.Time
}

// DeriveAgentID derives a stable AgentID from an Ed25519 public key.
// Format: "mh_" + first 16 hex chars of SHA-256(publicKey).
func DeriveAgentID(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return AgentIDPrefix + hex.EncodeToString(hash[:])[:16]
}

// NewAgentIdentity generates a new agent identity with a fresh Ed25519 keypair.
func NewAgentIdentity(name string) (*AgentIdentity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &AgentIdentity{
		AgentID:    DeriveAgentID(pub),
		AgentName:  name,
		PublicKey:  pub,
		PrivateKey: priv,
		CreatedAt:  time.Now(),
	}, nil
}

// LoadFromKeys reconstructs an agent identity from stored keys.
func LoadFromKeys(name string, priv ed25519.PrivateKey, createdAt time.Time) (*AgentIdentity, error) {
	pub := priv.Public().(ed25519.PublicKey)
	return &AgentIdentity{
		AgentID:    DeriveAgentID(pub),
		AgentName:  name,
		PublicKey:  pub,
		PrivateKey: priv,
		CreatedAt:  createdAt,
	}, nil
}

// Certificate returns the agent's self-signed certificate.
func (ai *AgentIdentity) Certificate() AgentCertificate {
	return AgentCertificate{
		AgentID:   ai.AgentID,
		AgentName: ai.AgentName,
		CreatedAt: ai.CreatedAt,
	}
}

// Sign signs a message using the agent's private key.
func (ai *AgentIdentity) Sign(message []byte) []byte {
	return ed25519.Sign(ai.PrivateKey, message)
}

// Verify verifies a signature against a message using the agent's public key.
func (ai *AgentIdentity) Verify(message, signature []byte) bool {
	return ed25519.Verify(ai.PublicKey, message, signature)
}

// PublicKeyBytes returns the raw bytes of the public key.
func (ai *AgentIdentity) PublicKeyBytes() []byte {
	return []byte(ai.PublicKey)
}
