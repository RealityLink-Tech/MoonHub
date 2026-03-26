package friends

import (
	"errors"
)

// ErrDuplicateRequest is returned when a friend request already exists.
var ErrDuplicateRequest = errors.New("friend request already exists")

// Manager handles the friend request/response lifecycle.
type Manager struct {
	store *Store
}

// NewManager creates a new friend manager backed by the given store.
func NewManager(store *Store) *Manager {
	return &Manager{store: store}
}

// SendRequest creates a pending friend request from another agent.
func (m *Manager) SendRequest(fromAgentID, fromAgentName string, publicKey []byte, message string) error {
	existing := m.store.GetFriend(fromAgentID)
	if existing != nil {
		return ErrDuplicateRequest
	}

	return m.store.AddFriend(&Friend{
		AgentID:   fromAgentID,
		AgentName: fromAgentName,
		PublicKey: publicKey,
		Status:    StatusPending,
		Message:   message,
	})
}

// AcceptRequest accepts a pending friend request.
func (m *Manager) AcceptRequest(agentID, myName string, myPublicKey []byte) (*Friend, error) {
	existing := m.store.GetFriend(agentID)
	if existing == nil {
		return nil, errors.New("friend request not found")
	}
	if existing.Status != StatusPending {
		return nil, errors.New("friend request is not pending")
	}

	if err := m.store.UpdateStatus(agentID, StatusAccepted); err != nil {
		return nil, err
	}

	return m.store.GetFriend(agentID), nil
}

// RejectRequest rejects a pending friend request.
func (m *Manager) RejectRequest(agentID string) (*Friend, error) {
	existing := m.store.GetFriend(agentID)
	if existing == nil {
		return nil, errors.New("friend request not found")
	}

	if err := m.store.UpdateStatus(agentID, StatusRejected); err != nil {
		return nil, err
	}

	return m.store.GetFriend(agentID), nil
}

// RevokeFriend removes an accepted friend entirely.
func (m *Manager) RevokeFriend(agentID string) error {
	return m.store.RemoveFriend(agentID)
}

// GetPendingRequests returns all pending friend requests.
func (m *Manager) GetPendingRequests() []*Friend {
	return m.store.ListFriends(StatusPending)
}

// ListAcceptedFriends returns all accepted friends.
func (m *Manager) ListAcceptedFriends() []*Friend {
	return m.store.ListFriends(StatusAccepted)
}
