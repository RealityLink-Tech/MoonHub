package transport

import (
	"crypto/ed25519"
	"fmt"
	"sync"

	_ "github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

type Manager struct {
	localID  string
	privKey  ed25519.PrivateKey
	mu       sync.RWMutex
	conns    map[string]*AgentConn
	resolver *Resolver
}

func NewManager(localID string, privKey ed25519.PrivateKey) *Manager {
	return &Manager{
		localID: localID,
		privKey: privKey,
		conns:   make(map[string]*AgentConn),
	}
}

func (m *Manager) LocalAgentID() string { return m.localID }

// SetResolver sets the connection resolver for LAN/Cloud path selection.
func (m *Manager) SetResolver(r *Resolver) {
	m.resolver = r
}

func (m *Manager) GetOrCreate(remoteAgentID, wsURL string) (*AgentConn, error) {
	m.mu.RLock()
	if conn, ok := m.conns[remoteAgentID]; ok {
		m.mu.RUnlock()
		return conn, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.conns[remoteAgentID]; ok {
		return conn, nil
	}

	// If no explicit URL, use resolver
	if wsURL == "" && m.resolver != nil {
		strategy, err := m.resolver.Resolve(remoteAgentID)
		if err != nil {
			return nil, err
		}
		wsURL = strategy.URL
	}

	if wsURL == "" {
		return nil, fmt.Errorf("no connection URL for agent %s", remoteAgentID)
	}

	conn := NewAgentConn(m.localID, remoteAgentID, m.privKey, wsURL)
	m.conns[remoteAgentID] = conn
	return conn, nil
}

func (m *Manager) Get(remoteAgentID string) *AgentConn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conns[remoteAgentID]
}

func (m *Manager) Remove(remoteAgentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.conns[remoteAgentID]; ok {
		conn.Close()
		delete(m.conns, remoteAgentID)
	}
}

func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.conns))
	for id := range m.conns {
		ids = append(ids, id)
	}
	return ids
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, conn := range m.conns {
		conn.Close()
		delete(m.conns, id)
	}
}
