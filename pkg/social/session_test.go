package social

import (
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	if sm == nil {
		t.Fatal("expected non-nil session manager")
	}
	if sm.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", sm.Count())
	}
}

func TestGetOrCreateSession(t *testing.T) {
	sm := NewSessionManager()

	deviceID := "test-device-123"

	// Create new session
	session1 := sm.GetOrCreateSession(deviceID)
	if session1 == nil {
		t.Fatal("expected non-nil session")
	}
	if session1.DeviceID != deviceID {
		t.Errorf("expected device ID %s, got %s", deviceID, session1.DeviceID)
	}
	if session1.SessionKey == "" {
		t.Error("expected non-empty session key")
	}
	if sm.Count() != 1 {
		t.Errorf("expected 1 session, got %d", sm.Count())
	}

	// Get existing session
	session2 := sm.GetOrCreateSession(deviceID)
	if session1.SessionKey != session2.SessionKey {
		t.Errorf("expected same session key, got %s vs %s", session1.SessionKey, session2.SessionKey)
	}
	if sm.Count() != 1 {
		t.Errorf("expected 1 session after get, got %d", sm.Count())
	}
}

func TestGetSession(t *testing.T) {
	sm := NewSessionManager()
	deviceID := "test-device-456"

	// Get non-existent session
	session := sm.GetSession(deviceID)
	if session != nil {
		t.Error("expected nil for non-existent session")
	}

	// Create and get session
	sm.GetOrCreateSession(deviceID)
	session = sm.GetSession(deviceID)
	if session == nil {
		t.Error("expected non-nil session")
	}
}

func TestRemoveSession(t *testing.T) {
	sm := NewSessionManager()
	deviceID := "test-device-789"

	// Create and remove session
	sm.GetOrCreateSession(deviceID)
	if sm.Count() != 1 {
		t.Errorf("expected 1 session, got %d", sm.Count())
	}

	sm.RemoveSession(deviceID)
	if sm.Count() != 0 {
		t.Errorf("expected 0 sessions after remove, got %d", sm.Count())
	}

	// Remove non-existent session (should not panic)
	sm.RemoveSession("non-existent")
}

func TestCleanupInactiveSessions(t *testing.T) {
	sm := NewSessionManager()

	// Create sessions
	deviceID1 := "device-1"
	deviceID2 := "device-2"

	sm.GetOrCreateSession(deviceID1)
	sm.GetOrCreateSession(deviceID2)

	// Manually set one session as inactive
	sm.mu.Lock()
	session := sm.sessions[GenerateSessionKey(deviceID1)]
	session.LastActive = time.Now().Add(-2 * time.Hour)
	sm.mu.Unlock()

	// Cleanup sessions inactive for more than 1 hour
	removed := sm.CleanupInactiveSessions(1 * time.Hour)
	if removed != 1 {
		t.Errorf("expected 1 session removed, got %d", removed)
	}
	if sm.Count() != 1 {
		t.Errorf("expected 1 session remaining, got %d", sm.Count())
	}

	// Verify correct session was removed
	if sm.GetSession(deviceID1) != nil {
		t.Error("expected device-1 session to be removed")
	}
	if sm.GetSession(deviceID2) == nil {
		t.Error("expected device-2 session to remain")
	}
}

func TestGenerateSessionKey(t *testing.T) {
	deviceID := "test-device-abc"

	key1 := GenerateSessionKey(deviceID)
	key2 := GenerateSessionKey(deviceID)

	// Same device ID should produce same key
	if key1 != key2 {
		t.Errorf("expected same keys for same device ID, got %s vs %s", key1, key2)
	}

	// Different device IDs should produce different keys
	key3 := GenerateSessionKey("different-device")
	if key1 == key3 {
		t.Error("expected different keys for different device IDs")
	}

	// Key should start with "lan:"
	if len(key1) < 4 || key1[:4] != "lan:" {
		t.Errorf("expected key to start with 'lan:', got %s", key1)
	}
}

func TestSessionKeyConsistency(t *testing.T) {
	// Test that session keys are consistent across multiple calls
	testCases := []string{
		"device-1",
		"device-2",
		"very-long-device-id-with-special-chars-!@#$%",
		"",
		"设备-中文", // Unicode test
	}

	for _, deviceID := range testCases {
		key1 := GenerateSessionKey(deviceID)
		key2 := GenerateSessionKey(deviceID)
		if key1 != key2 {
			t.Errorf("inconsistent keys for deviceID %q: %s vs %s", deviceID, key1, key2)
		}
	}
}
