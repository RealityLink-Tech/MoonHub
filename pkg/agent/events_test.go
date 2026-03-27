package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentEventSerialization(t *testing.T) {
	evt := AgentEvent{
		Kind:       EventToolStart,
		ToolName:   "web_search",
		ToolArgs:   map[string]any{"query": "test"},
		Content:    "",
		SessionKey: "agent:main:pico:direct:pico:abc",
		Channel:    "pico",
		ChatID:     "pico:abc",
		Iteration:  1,
	}

	data, err := json.Marshal(evt)
	assert.NoError(t, err)

	var decoded AgentEvent
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, EventToolStart, decoded.Kind)
	assert.Equal(t, "web_search", decoded.ToolName)
	assert.Equal(t, "pico", decoded.Channel)
	assert.Equal(t, 1, decoded.Iteration)
	// ToolArgs with map[string]any should survive round-trip
	assert.Equal(t, "test", decoded.ToolArgs["query"])
}

func TestAgentEventContentOnly(t *testing.T) {
	evt := AgentEvent{
		Kind:    EventContentChunk,
		Content: "hello world",
		Channel: "pico",
		ChatID:  "pico:abc",
	}

	data, err := json.Marshal(evt)
	assert.NoError(t, err)

	// Verify tool fields are omitted when empty
	assert.NotContains(t, string(data), "tool_name")
	assert.NotContains(t, string(data), "tool_args")

	var decoded AgentEvent
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", decoded.Content)
	assert.Equal(t, "", decoded.ToolName)
}

func TestAgentEventDone(t *testing.T) {
	evt := AgentEvent{
		Kind:    EventDone,
		Content: "final response text",
		Channel: "pico",
		ChatID:  "pico:abc",
	}

	data, err := json.Marshal(evt)
	assert.NoError(t, err)

	var decoded AgentEvent
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, EventDone, decoded.Kind)
	assert.Equal(t, "final response text", decoded.Content)
}

func TestEventEmitterNilSafe(t *testing.T) {
	var emitter EventEmitter
	// Should not panic when emitter is nil
	emitter.Emit(AgentEvent{Kind: EventToolStart, Channel: "pico", ChatID: "pico:x"})
}

func TestEventEmitterFiresCallback(t *testing.T) {
	var received AgentEvent
	emitter := EventEmitter(func(evt AgentEvent) { received = evt })
	emitter.Emit(AgentEvent{Kind: EventToolStart, Channel: "pico", ChatID: "pico:x"})
	assert.Equal(t, EventToolStart, received.Kind)
	assert.Equal(t, "pico", received.Channel)
	assert.Equal(t, "pico:x", received.ChatID)
}
