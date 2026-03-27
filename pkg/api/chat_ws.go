package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/gorilla/websocket"
)

const (
	consoleChannel = "console"
	wsWriteTimeout = 10
	// TODO: add ping/pong keepalive for production use
	// See https://github.com/gorilla/websocket/blob/master/examples/chat/conn.go
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsClient represents a single WebSocket connection.
type wsClient struct {
	conn     *websocket.Conn
	sendCh   chan []byte
	done     chan struct{}
	doneOnce sync.Once
}

// ChatHub manages active WebSocket connections and broadcasts agent events.
type ChatHub struct {
	mu      sync.RWMutex
	clients map[string]*wsClient // sessionID -> client
	agent   *agent.AgentLoop
	// onEvent is called for each event sent to a client (used in tests).
	onEvent func(clientID string, evt agent.AgentEvent)
}

func newChatHub(agent *agent.AgentLoop) *ChatHub {
	return &ChatHub{
		clients: make(map[string]*wsClient),
		agent:   agent,
	}
}

// SetAgent updates the agent loop reference.
func (h *ChatHub) SetAgent(agent *agent.AgentLoop) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agent = agent
}

// AddClient registers a new WebSocket client.
func (h *ChatHub) AddClient(sessionID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing connection for this session if any
	if old, exists := h.clients[sessionID]; exists {
		old.doneOnce.Do(func() { close(old.done) })
		old.conn.Close()
	}

	client := &wsClient{
		conn:   conn,
		sendCh: make(chan []byte, 64),
		done:   make(chan struct{}),
	}
	h.clients[sessionID] = client

	go h.writePump(sessionID, client)
}

// RemoveClient removes a WebSocket client.
func (h *ChatHub) RemoveClient(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client, exists := h.clients[sessionID]; exists {
		client.doneOnce.Do(func() { close(client.done) })
		delete(h.clients, sessionID)
	}
}

// Broadcast sends an agent event to the matching WebSocket client.
// It matches on Channel=="console" and ChatID==sessionID.
func (h *ChatHub) Broadcast(evt bus.AgentEvent) {
	if evt.Channel != consoleChannel {
		return
	}

	h.mu.RLock()
	client, exists := h.clients[evt.ChatID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	// Map agent event kind to WebSocket message type
	msgType := "agent." + string(evt.Kind)
	payload := map[string]any{}

	switch evt.Kind {
	case bus.EventToolStart:
		payload["tool_name"] = evt.ToolName
		if argsJSON, err := json.Marshal(evt.ToolArgs); err == nil {
			if len(argsJSON) > 100 {
				payload["tool_args_preview"] = string(argsJSON[:100]) + "..."
			} else {
				payload["tool_args_preview"] = string(argsJSON)
			}
		}
	case bus.EventToolEnd:
		payload["tool_name"] = evt.ToolName
		payload["success"] = evt.ToolError == ""
		payload["error"] = evt.ToolError
	case bus.EventContentChunk:
		payload["content"] = evt.Content
		payload["done"] = false
	case bus.EventContentStart:
		payload["done"] = false
	case bus.EventThinking:
		payload["content"] = evt.Content
	case bus.EventDone:
		payload["content"] = evt.Content
	case bus.EventError:
		payload["message"] = evt.Content
	default:
		return
	}

	msg := map[string]any{
		"type":      msgType,
		"timestamp": evt.Iteration,
		"payload":   payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	if h.onEvent != nil {
		h.onEvent(evt.ChatID, evt)
	}

	select {
	case client.sendCh <- data:
	case <-client.done:
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
func (h *ChatHub) writePump(sessionID string, client *wsClient) {
	defer func() {
		client.conn.Close()
		h.RemoveClient(sessionID)
	}()

	for {
		select {
		case msg, ok := <-client.sendCh:
			if !ok {
				return
			}
			client.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout * time.Second))
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-client.done:
			return
		}
	}
}

// HandleMessage processes an incoming chat message from a WebSocket client.
func (h *ChatHub) HandleMessage(sessionID, content string) {
	h.mu.RLock()
	agent := h.agent
	h.mu.RUnlock()

	if agent == nil {
		return
	}

	sessionKey := consoleChannel + ":" + sessionID
	go func() {
		_, err := agent.ProcessDirectWithChannel(
			context.Background(),
			content,
			sessionKey,
			consoleChannel,
			sessionID,
		)
		if err != nil {
			log.Printf("[api] ProcessDirectWithChannel failed: %v", err)
		}
	}()
}

// ==================== HTTP Handler ====================

func (h *Handler) handleChatWS(w http.ResponseWriter, r *http.Request) {
	// Validate token from query param
	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing token")
		return
	}

	deviceID, ok := h.tokenStore.Validate(token)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	_ = deviceID // available for future multi-device management

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[api] WebSocket upgrade failed: %v", err)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = "default"
	}

	hub := h.chatHub
	if hub == nil {
		conn.Close()
		return
	}

	hub.AddClient(sessionID, conn)

	// Read pump -- handle incoming messages
	defer func() {
		hub.RemoveClient(sessionID)
	}()

	conn.SetReadLimit(1 << 20) // 1 MB max message size
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		if msg.Type == "message.send" && msg.Content != "" {
			hub.HandleMessage(sessionID, msg.Content)
		}
	}
}
