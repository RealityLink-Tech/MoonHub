package mhp

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// EnvelopeWriter writes signed mhp Envelopes to a WebSocket connection with write locking.
type EnvelopeWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// NewEnvelopeWriter creates a new EnvelopeWriter for the given WebSocket connection.
func NewEnvelopeWriter(conn *websocket.Conn) (*EnvelopeWriter, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}
	return &EnvelopeWriter{conn: conn}, nil
}

// WriteEnvelope marshals and sends an Envelope as a JSON text message.
// Thread-safe: uses a mutex to prevent concurrent writes.
func (w *EnvelopeWriter) WriteEnvelope(env *Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	return w.conn.WriteMessage(websocket.TextMessage, data)
}

// EnvelopeReader reads mhp Envelopes from a WebSocket connection.
type EnvelopeReader struct {
	conn *websocket.Conn
}

// NewEnvelopeReader creates a new EnvelopeReader for the given WebSocket connection.
func NewEnvelopeReader(conn *websocket.Conn) *EnvelopeReader {
	return &EnvelopeReader{conn: conn}
}

// ReadEnvelope reads the next message from the WebSocket and unmarshals it as an Envelope.
// Returns an error if the message is invalid JSON or the connection is closed.
func (r *EnvelopeReader) ReadEnvelope() (*Envelope, error) {
	_, data, err := r.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("invalid envelope JSON: %w", err)
	}

	return &env, nil
}
