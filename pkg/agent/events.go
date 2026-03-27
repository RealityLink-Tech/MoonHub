package agent

import "github.com/RealityLink-Tech/MoonHub/pkg/bus"

// AgentEventKind identifies the type of agent activity event.
type AgentEventKind = bus.AgentEventKind

// AgentEvent represents a real-time event emitted during agent processing.
// It is designed to be JSON-serializable and transport-agnostic.
type AgentEvent = bus.AgentEvent

// Event kind constants — re-exported from bus for convenience.
const (
	EventToolStart    = bus.EventToolStart
	EventToolEnd      = bus.EventToolEnd
	EventContentStart = bus.EventContentStart
	EventContentChunk = bus.EventContentChunk
	EventThinking     = bus.EventThinking
	EventDone         = bus.EventDone
	EventError        = bus.EventError
)

// EventEmitter is a callback function that receives agent events.
// It is nil-safe: calling a nil EventEmitter is a no-op.
type EventEmitter func(AgentEvent)

// Emit calls the event emitter if it is non-nil.
func (e EventEmitter) Emit(evt AgentEvent) {
	if e != nil {
		e(evt)
	}
}
