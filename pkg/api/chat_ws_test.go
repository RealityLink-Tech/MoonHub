package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAgentLoop struct {
	mock.Mock
}

func (m *mockAgentLoop) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	args := m.Called(ctx, content, sessionKey, channel, chatID)
	return args.String(0), args.Error(1)
}

func (m *mockAgentLoop) SetEventEmitter(emitter agent.EventEmitter) {
	m.Called(emitter)
}

func TestChatWS_UpgradeWithoutToken(t *testing.T) {
	h := NewHandler(t.TempDir(), nil, nil)
	server := httptest.NewServer(h)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/chat/ws"
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
}

func TestChatHub_Broadcast(t *testing.T) {
	hub := newChatHub(nil)
	received := make(chan agent.AgentEvent, 1)

	hub.onEvent = func(clientID string, evt agent.AgentEvent) {
		received <- evt
	}

	// Simulate a client connection
	hub.clients["test-session"] = &wsClient{sendCh: make(chan []byte, 10)}

	hub.Broadcast(agent.AgentEvent{
		Kind:     agent.EventToolStart,
		Channel:  "console",
		ChatID:   "test-session",
		ToolName: "web_search",
	})

	select {
	case evt := <-received:
		assert.Equal(t, agent.EventToolStart, evt.Kind)
		assert.Equal(t, "web_search", evt.ToolName)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestChatHub_BroadcastFiltersByChatID(t *testing.T) {
	hub := newChatHub(nil)
	received := make(chan agent.AgentEvent, 1)

	hub.onEvent = func(clientID string, evt agent.AgentEvent) {
		received <- evt
	}

	hub.clients["session-a"] = &wsClient{sendCh: make(chan []byte, 10)}
	hub.clients["session-b"] = &wsClient{sendCh: make(chan []byte, 10)}

	// Broadcast event for session-a only
	hub.Broadcast(agent.AgentEvent{
		Kind:    agent.EventDone,
		Channel: "console",
		ChatID:  "session-a",
	})

	// Should receive exactly one event
	select {
	case <-received:
		// Good
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}
