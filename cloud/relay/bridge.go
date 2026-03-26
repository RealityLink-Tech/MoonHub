// cloud/relay/bridge.go
package relay

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Bridge manages agent WebSocket connections and bridges.
type Bridge struct {
	mu       sync.RWMutex
	agents   map[string]*websocket.Conn
	commands chan bridgeCommand
	done     chan struct{}
}

type bridgeCommand struct {
	from    string
	conn    *websocket.Conn
	message []byte
}

// NewBridge creates a new relay bridge.
func NewBridge() *Bridge {
	return &Bridge{
		agents:   make(map[string]*websocket.Conn),
		commands: make(chan bridgeCommand, 256),
		done:     make(chan struct{}),
	}
}

// Run starts the bridge event loop. Blocks until Stop is called.
func (b *Bridge) Run() {
	for {
		select {
		case cmd := <-b.commands:
			b.handleCommand(cmd)
		case <-b.done:
			return
		}
	}
}

// Stop shuts down the bridge.
func (b *Bridge) Stop() {
	close(b.done)
}

// RegisterAgent registers an agent connection.
func (b *Bridge) RegisterAgent(agentID string, conn *websocket.Conn) {
	b.mu.Lock()
	if old, ok := b.agents[agentID]; ok {
		old.Close()
	}
	b.agents[agentID] = conn
	b.mu.Unlock()
}

// UnregisterAgent removes an agent connection.
func (b *Bridge) UnregisterAgent(agentID string) {
	b.mu.Lock()
	delete(b.agents, agentID)
	b.mu.Unlock()
}

// HandleConnection processes incoming messages from an agent connection.
func (b *Bridge) HandleConnection(agentID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		b.UnregisterAgent(agentID)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		b.commands <- bridgeCommand{
			from:    agentID,
			conn:    conn,
			message: msg,
		}
	}
}

func (b *Bridge) handleCommand(cmd bridgeCommand) {
	msg := string(cmd.message)

	if strings.HasPrefix(msg, "CONNECT ") {
		targetID := strings.TrimPrefix(msg, "CONNECT ")
		b.handleConnect(cmd.from, cmd.conn, targetID)
		return
	}

	if msg == "PING" {
		cmd.conn.WriteMessage(websocket.TextMessage, []byte("PONG"))
		return
	}
}

func (b *Bridge) handleConnect(fromID string, fromConn *websocket.Conn, targetID string) {
	b.mu.RLock()
	targetConn, ok := b.agents[targetID]
	b.mu.RUnlock()

	if !ok {
		fromConn.WriteMessage(websocket.TextMessage, []byte("ERROR target_offline"))
		return
	}

	fromConn.WriteMessage(websocket.TextMessage, []byte("CONNECTED"))
	targetConn.WriteMessage(websocket.TextMessage, []byte("CONNECTED"))

	go b.forward(fromID, targetID, fromConn, targetConn)
	go b.forward(targetID, fromID, targetConn, fromConn)
}

func (b *Bridge) forward(fromID, toID string, src, dst *websocket.Conn) {
	defer func() {
		dst.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR peer_disconnected")))
	}()

	for {
		mt, msg, err := src.ReadMessage()
		if err != nil {
			return
		}
		msgStr := string(msg)
		if strings.HasPrefix(msgStr, "CONNECT ") || msgStr == "PING" || msgStr == "PONG" {
			continue
		}
		if err := dst.WriteMessage(mt, msg); err != nil {
			return
		}
	}
}

// ServeHTTP implements http.Handler for the relay WebSocket endpoint.
func (b *Bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		conn.Close()
		return
	}

	b.RegisterAgent(agentID, conn)
	go b.HandleConnection(agentID, conn)
}
