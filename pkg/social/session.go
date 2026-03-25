package social

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// SessionManager manages chat sessions for LAN clients.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// Session represents a client session.
type Session struct {
	// DeviceID is the unique identifier of the paired device.
	DeviceID string
	// SessionKey is the key used to identify this session in the agent.
	SessionKey string
	// LastActive is the last time this session was active.
	LastActive time.Time
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreateSession returns an existing session or creates a new one.
// The sessionKey format is "lan:<deviceID_hash>" to isolate conversations per device.
func (sm *SessionManager) GetOrCreateSession(deviceID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionKey := GenerateSessionKey(deviceID)

	// Check if session exists
	if session, ok := sm.sessions[sessionKey]; ok {
		session.LastActive = time.Now()
		return session
	}

	// Create new session
	session := &Session{
		DeviceID:   deviceID,
		SessionKey: sessionKey,
		LastActive: time.Now(),
	}
	sm.sessions[sessionKey] = session
	return session
}

// GetSession retrieves a session by device ID.
func (sm *SessionManager) GetSession(deviceID string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionKey := GenerateSessionKey(deviceID)
	return sm.sessions[sessionKey]
}

// RemoveSession removes a session by device ID.
func (sm *SessionManager) RemoveSession(deviceID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionKey := GenerateSessionKey(deviceID)
	delete(sm.sessions, sessionKey)
}

// CleanupInactiveSessions removes sessions that have been inactive for longer than the specified duration.
func (sm *SessionManager) CleanupInactiveSessions(maxInactiveDuration time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, session := range sm.sessions {
		if now.Sub(session.LastActive) > maxInactiveDuration {
			delete(sm.sessions, key)
			removed++
		}
	}

	return removed
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// GenerateSessionKey creates a session key from device ID.
// The format is "lan:<sha256_hash_of_deviceID>" to ensure unique but consistent keys.
func GenerateSessionKey(deviceID string) string {
	hash := sha256.Sum256([]byte(deviceID))
	hashStr := hex.EncodeToString(hash[:])[:16] // Use first 16 chars for readability
	return "lan:" + hashStr
}
