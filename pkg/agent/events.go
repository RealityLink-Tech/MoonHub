package agent

// AgentEventKind identifies the type of agent activity event.
type AgentEventKind string

const (
	EventToolStart    AgentEventKind = "tool_start"
	EventToolEnd      AgentEventKind = "tool_end"
	EventContentStart AgentEventKind = "content_start"
	EventContentChunk AgentEventKind = "content_chunk"
	EventThinking     AgentEventKind = "thinking"
	EventDone         AgentEventKind = "done"
	EventError        AgentEventKind = "error"
)

// AgentEvent represents a real-time event emitted during agent processing.
// It is designed to be JSON-serializable and transport-agnostic.
type AgentEvent struct {
	Kind       AgentEventKind `json:"kind"`
	ToolName   string         `json:"tool_name,omitempty"`
	ToolArgs   map[string]any `json:"tool_args,omitempty"`
	ToolError  string         `json:"tool_error,omitempty"`
	Content    string         `json:"content,omitempty"`
	SessionKey string         `json:"session_key"`
	Channel    string         `json:"channel"`
	ChatID     string         `json:"chat_id"`
	Iteration  int            `json:"iteration,omitempty"`
}

// EventEmitter is a callback function that receives agent events.
// It is nil-safe: calling a nil EventEmitter is a no-op.
type EventEmitter func(AgentEvent)

// Emit calls the event emitter if it is non-nil.
func (e EventEmitter) Emit(evt AgentEvent) {
	if e != nil {
		e(evt)
	}
}
