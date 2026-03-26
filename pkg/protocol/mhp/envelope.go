package mhp

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProtocolVersion is the current mhp protocol version.
const ProtocolVersion = 1

// MaxMessageAge is the maximum age of a message before it's considered stale (5 minutes).
const MaxMessageAge = 5 * time.Minute

// dedupWindow is how long to remember message IDs for replay detection.
const dedupWindow = 10 * time.Minute

// Envelope is the signed message envelope for all mhp messages.
type Envelope struct {
	Version   int             `json:"version"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Type      MessageType     `json:"type"`
	ID        string          `json:"id"`
	Timestamp int64           `json:"timestamp"`
	Signature string          `json:"signature,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// NewEnvelope creates a new unsigned envelope.
func NewEnvelope(from, to string, msgType MessageType, payload any) *Envelope {
	var rawPayload json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			rawPayload = json.RawMessage(`{}`)
		} else {
			rawPayload = data
		}
	}

	return &Envelope{
		Version:   ProtocolVersion,
		From:      from,
		To:        to,
		Type:      msgType,
		ID:        uuid.New().String(),
		Timestamp: time.Now().Unix(),
		Payload:   rawPayload,
	}
}

// Sign signs the envelope using the given private key and sets the Signature field.
func (e *Envelope) Sign(priv ed25519.PrivateKey) (string, error) {
	message := e.signedFields()
	sig := ed25519.Sign(priv, []byte(message))
	e.Signature = base64.StdEncoding.EncodeToString(sig)
	return e.Signature, nil
}

// Verify verifies the envelope's signature using the given public key.
func (e *Envelope) Verify(pub ed25519.PublicKey) error {
	if e.Signature == "" {
		return fmt.Errorf("envelope has no signature")
	}

	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	message := e.signedFields()
	if !ed25519.Verify(pub, []byte(message), sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// signedFields returns the canonical string that is signed.
func (e *Envelope) signedFields() string {
	payload := "{}"
	if e.Payload != nil {
		trimmed := strings.TrimSpace(string(e.Payload))
		if trimmed != "" && trimmed != "null" {
			payload = trimmed
		}
	}
	return fmt.Sprintf("%d%s%s%s%s%d%s",
		e.Version, e.From, e.To, e.Type, e.ID, e.Timestamp, payload)
}

// DecodePayload decodes the envelope's payload into the given type.
func DecodePayload[T any](e *Envelope) (*T, error) {
	var result T
	if e.Payload == nil {
		return nil, fmt.Errorf("envelope has no payload")
	}
	if err := json.Unmarshal(e.Payload, &result); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return &result, nil
}

// MessageDedup detects replay attacks by tracking seen message IDs.
type MessageDedup struct {
	mu      sync.RWMutex
	seen    map[string]time.Time
	cleanup time.Time
}

// NewMessageDedup creates a new message deduplication checker.
func NewMessageDedup() *MessageDedup {
	return &MessageDedup{
		seen:    make(map[string]time.Time),
		cleanup: time.Now(),
	}
}

// CheckAndAdd checks if a message is a duplicate or stale.
// Returns true if the message is fresh and should be processed.
func (d *MessageDedup) CheckAndAdd(msgID string, timestamp int64) bool {
	now := time.Now()
	msgTime := time.Unix(timestamp, 0)

	if now.Sub(msgTime) > MaxMessageAge {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if now.Sub(d.cleanup) > time.Minute {
		for id, seenAt := range d.seen {
			if now.Sub(seenAt) > dedupWindow {
				delete(d.seen, id)
			}
		}
		d.cleanup = now
	}

	if _, exists := d.seen[msgID]; exists {
		return false
	}

	d.seen[msgID] = now
	return true
}
