// Package social provides social/communication primitives for MoonHub.
package social

import "time"

// MessageType defines the type of message.
type MessageType string

const (
	// MessageTypeText is a regular text message.
	MessageTypeText MessageType = "text"
	// MessageTypeCommand is a command message.
	MessageTypeCommand MessageType = "command"
)

// ChatRequest represents an incoming chat message from a LAN client.
type ChatRequest struct {
	// Type is the message type (text or command).
	Type MessageType `json:"type"`
	// Content is the message content.
	Content string `json:"content"`
}

// ChatResponse represents a response message to a LAN client.
type ChatResponse struct {
	// ID is the unique message identifier.
	ID string `json:"id"`
	// Content is the response content.
	Content string `json:"content"`
	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
	// Error is an optional error message.
	Error string `json:"error,omitempty"`
}

// SSEEvent represents a Server-Sent Event for streaming responses.
type SSEEvent struct {
	// Event is the event type: "start", "chunk", "end", "error".
	Event string `json:"event"`
	// Data is the event payload.
	Data interface{} `json:"data"`
}

// SSEStartData is the data payload for the "start" event.
type SSEStartData struct {
	// ID is the unique message identifier.
	ID string `json:"id"`
}

// SSEChunkData is the data payload for the "chunk" event.
type SSEChunkData struct {
	// Content is the text chunk.
	Content string `json:"content"`
}

// SSEEndData is the data payload for the "end" event.
type SSEEndData struct {
	// Content is the complete response content.
	Content string `json:"content"`
	// Timestamp is when the response completed.
	Timestamp time.Time `json:"timestamp"`
}

// SSEErrorData is the data payload for the "error" event.
type SSEErrorData struct {
	// Error is the error message.
	Error string `json:"error"`
}

// DeviceInfo represents device information for LAN API.
type DeviceInfo struct {
	// ID is the unique device identifier.
	ID string `json:"id"`
	// Name is the friendly device name.
	Name string `json:"name"`
	// Status is the device status (e.g., "online", "offline").
	Status string `json:"status"`
	// Version is the software version.
	Version string `json:"version"`
}
