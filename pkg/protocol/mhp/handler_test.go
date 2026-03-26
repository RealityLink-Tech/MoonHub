package mhp

import (
	"crypto/ed25519"
	"sync"
	"testing"
)

// mockFriendStore implements HandlerFriendStore for testing.
type mockFriendStore struct {
	mu      sync.Mutex
	friends map[string]mockFriendEntry
}

type mockFriendEntry struct {
	publicKey []byte
	status    string
}

func newMockFriendStore() *mockFriendStore {
	return &mockFriendStore{friends: make(map[string]mockFriendEntry)}
}

func (s *mockFriendStore) IsFriend(agentID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.friends[agentID]
	return ok && f.status == "accepted"
}

func (s *mockFriendStore) GetPublicKey(agentID string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.friends[agentID]
	if !ok {
		return nil
	}
	return f.publicKey
}

func (s *mockFriendStore) SetFriend(agentID string, publicKey []byte, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.friends[agentID] = mockFriendEntry{publicKey: publicKey, status: status}
}

func TestHandler_FriendRequest_NewFriend(t *testing.T) {
	_, remotePriv, _ := ed25519.GenerateKey(nil)

	store := newMockFriendStore()
	var received FriendRequestPayload

	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
		OnFriendRequest: func(env *Envelope, payload *FriendRequestPayload) {
			received = *payload
		},
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "RemoteAgent",
		Message:   "Hi there!",
	})
	_, _ = env.Sign(remotePriv)

	result := handler.HandleEnvelope(env)
	if result != HandlerAccepted {
		t.Errorf("expected HandlerAccepted, got %d", result)
	}

	if received.AgentName != "RemoteAgent" {
		t.Errorf("expected AgentName RemoteAgent, got %s", received.AgentName)
	}
	if received.Message != "Hi there!" {
		t.Errorf("expected Message 'Hi there!', got %s", received.Message)
	}
}

func TestHandler_FriendRequest_Duplicate(t *testing.T) {
	_, remotePriv, _ := ed25519.GenerateKey(nil)
	remotePub := remotePriv.Public().(ed25519.PublicKey)

	store := newMockFriendStore()
	store.SetFriend("mh_remote0000000000", remotePub, "pending")

	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "RemoteAgent",
	})
	_, _ = env.Sign(remotePriv)

	result := handler.HandleEnvelope(env)
	if result != HandlerRejected {
		t.Errorf("expected HandlerRejected for duplicate, got %d", result)
	}
}

func TestHandler_FriendRequest_InvalidSignature(t *testing.T) {
	_, badPriv, _ := ed25519.GenerateKey(nil)

	store := newMockFriendStore()
	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "RemoteAgent",
	})
	_, _ = env.Sign(badPriv)

	result := handler.HandleEnvelope(env)
	if result != HandlerRejected {
		t.Errorf("expected HandlerRejected for unknown sender, got %d", result)
	}
}

func TestHandler_FriendAccept_FromPending(t *testing.T) {
	_, remotePriv, _ := ed25519.GenerateKey(nil)
	remotePub := remotePriv.Public().(ed25519.PublicKey)

	store := newMockFriendStore()
	store.SetFriend("mh_remote0000000000", remotePub, "pending")

	accepted := false
	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
		OnFriendAccept: func(env *Envelope, payload *FriendAcceptPayload) {
			accepted = true
		},
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendAccept, &FriendAcceptPayload{
		AgentName: "RemoteAgent",
		PublicKey: remotePub,
	})
	_, _ = env.Sign(remotePriv)

	result := handler.HandleEnvelope(env)
	if result != HandlerAccepted {
		t.Errorf("expected HandlerAccepted, got %d", result)
	}
	if !accepted {
		t.Error("expected OnFriendAccept callback to be called")
	}
}

func TestHandler_UnknownType(t *testing.T) {
	store := newMockFriendStore()
	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", "unknown_type", nil)

	result := handler.HandleEnvelope(env)
	if result != HandlerRejected {
		t.Errorf("expected HandlerRejected for unknown type, got %d", result)
	}
}

func TestHandler_UnsignedEnvelope(t *testing.T) {
	store := newMockFriendStore()
	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "RemoteAgent",
	})

	result := handler.HandleEnvelope(env)
	if result != HandlerRejected {
		t.Errorf("expected HandlerRejected for unsigned envelope, got %d", result)
	}
}

func TestHandler_Dedup(t *testing.T) {
	_, remotePriv, _ := ed25519.GenerateKey(nil)

	store := newMockFriendStore()
	callCount := 0

	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
		OnFriendRequest: func(env *Envelope, payload *FriendRequestPayload) {
			callCount++
		},
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "RemoteAgent",
	})
	_, _ = env.Sign(remotePriv)

	result1 := handler.HandleEnvelope(env)
	if result1 != HandlerAccepted {
		t.Errorf("first call: expected HandlerAccepted, got %d", result1)
	}

	result2 := handler.HandleEnvelope(env)
	if result2 != HandlerDeduped {
		t.Errorf("second call: expected HandlerDeduped, got %d", result2)
	}

	if callCount != 1 {
		t.Errorf("expected callback called once, got %d", callCount)
	}
}

func TestHandler_StaleEnvelope(t *testing.T) {
	_, remotePriv, _ := ed25519.GenerateKey(nil)

	store := newMockFriendStore()
	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
	})

	env := NewEnvelope("mh_remote0000000000", "mh_local0000000000", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "RemoteAgent",
	})
	env.Timestamp = 0
	_, _ = env.Sign(remotePriv)

	result := handler.HandleEnvelope(env)
	if result != HandlerRejected {
		t.Errorf("expected HandlerRejected for stale envelope, got %d", result)
	}
}

func TestNewHandler_NilCallbacks(t *testing.T) {
	store := newMockFriendStore()
	handler := NewHandler(HandlerConfig{
		AgentID:     "mh_local0000000000",
		FriendStore: store,
	})
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}
