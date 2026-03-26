// pkg/transport/resolver.go
package transport

import (
	"fmt"
	"sync"
)

// ConnectionMode indicates whether a connection is LAN-direct or cloud-relayed.
type ConnectionMode int

const (
	ModeLAN   ConnectionMode = iota
	ModeCloud
)

// ConnectionStrategy describes how to reach a remote agent.
type ConnectionStrategy struct {
	Mode ConnectionMode
	URL  string
}

// CloudLookup provides agent lookup capability.
type CloudLookup interface {
	LookupAgent(agentID string) (*AgentLookupResult, error)
}

// Resolver determines the best connection strategy for reaching a remote agent.
type Resolver struct {
	cloud  CloudLookup
	mu     sync.RWMutex
	lanMap map[string]string // agentID -> direct WebSocket URL
}

// NewResolver creates a new Resolver.
func NewResolver(cloud CloudLookup) *Resolver {
	return &Resolver{
		cloud:  cloud,
		lanMap: make(map[string]string),
	}
}

// SetLANEndpoint registers a LAN-discovered endpoint for an agent.
func (r *Resolver) SetLANEndpoint(agentID, wsURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lanMap[agentID] = wsURL
}

// RemoveLANEndpoint removes a LAN-discovered endpoint.
func (r *Resolver) RemoveLANEndpoint(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lanMap, agentID)
}

// Resolve returns the best connection strategy for reaching a remote agent.
// Priority: LAN direct > Cloud relay > Offline.
func (r *Resolver) Resolve(agentID string) (*ConnectionStrategy, error) {
	// 1. Check LAN cache
	r.mu.RLock()
	lanURL, ok := r.lanMap[agentID]
	r.mu.RUnlock()

	if ok {
		return &ConnectionStrategy{Mode: ModeLAN, URL: lanURL}, nil
	}

	// 2. Check cloud directory
	if r.cloud != nil {
		result, err := r.cloud.LookupAgent(agentID)
		if err == nil && result.Online {
			return &ConnectionStrategy{Mode: ModeCloud, URL: result.RelayEndpoint}, nil
		}
	}

	return nil, fmt.Errorf("agent %s is offline", agentID)
}
