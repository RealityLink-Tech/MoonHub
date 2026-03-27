package agent

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockEmitter tracks received events
type mockEmitter struct {
	mock.Mock
	mu   sync.Mutex
	evts []AgentEvent
}

func (m *mockEmitter) Emit(evt AgentEvent) {
	m.mu.Lock()
	m.evts = append(m.evts, evt)
	m.mu.Unlock()
	m.Called(evt)
}

func (m *mockEmitter) Events() []AgentEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.evts
}

func TestEmitEvent_NilEmitter(t *testing.T) {
	al := &AgentLoop{}
	// Must not panic
	al.emitEvent(AgentEvent{Kind: EventToolStart})
}

func TestEmitEvent_FiresCallback(t *testing.T) {
	me := &mockEmitter{}
	me.On("Emit", mock.Anything).Maybe()
	al := &AgentLoop{eventEmitter: me.Emit}

	al.emitEvent(AgentEvent{Kind: EventToolStart, ToolName: "web_search"})
	assert.Len(t, me.Events(), 1)
	assert.Equal(t, EventToolStart, me.Events()[0].Kind)
	assert.Equal(t, "web_search", me.Events()[0].ToolName)
	me.AssertExpectations(t)
}
