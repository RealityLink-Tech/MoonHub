package friends

import (
	"testing"
)

func testDBPath(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/friends.db"
}

func TestManager_SendFriendRequest(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	err := mgr.SendRequest("mh_sender11111111", "TestSender", []byte("sender-key"), "Let's collaborate!")
	if err != nil {
		t.Fatal(err)
	}

	f := store.GetFriend("mh_sender11111111")
	if f == nil {
		t.Fatal("expected friend record")
	}
	if f.Status != StatusPending {
		t.Errorf("expected pending, got %s", f.Status)
	}
	if f.Message != "Let's collaborate!" {
		t.Errorf("expected message, got %s", f.Message)
	}
}

func TestManager_SendFriendRequest_Duplicate(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_ = mgr.SendRequest("mh_sender11111111", "Test", nil, "hi")
	err := mgr.SendRequest("mh_sender11111111", "Test", nil, "hi again")
	if err == nil {
		t.Error("expected error for duplicate request")
	}
	if err != ErrDuplicateRequest {
		t.Errorf("expected ErrDuplicateRequest, got %v", err)
	}
}

func TestManager_AcceptFriendRequest(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_ = mgr.SendRequest("mh_sender11111111", "TestSender", []byte("sender-key"), "hi")

	friend, err := mgr.AcceptRequest("mh_sender11111111", "MyAgent", []byte("my-key"))
	if err != nil {
		t.Fatal(err)
	}
	if friend.Status != StatusAccepted {
		t.Errorf("expected accepted, got %s", friend.Status)
	}
}

func TestManager_AcceptFriendRequest_NotFound(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_, err := mgr.AcceptRequest("mh_nonexistent", "Me", nil)
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestManager_RejectFriendRequest(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_ = mgr.SendRequest("mh_sender11111111", "Test", nil, "hi")

	friend, err := mgr.RejectRequest("mh_sender11111111")
	if err != nil {
		t.Fatal(err)
	}
	if friend.Status != StatusRejected {
		t.Errorf("expected rejected, got %s", friend.Status)
	}
}

func TestManager_RevokeFriend(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_ = mgr.SendRequest("mh_sender11111111", "Test", nil, "hi")
	_, _ = mgr.AcceptRequest("mh_sender11111111", "Me", nil)

	err := mgr.RevokeFriend("mh_sender11111111")
	if err != nil {
		t.Fatal(err)
	}

	f := store.GetFriend("mh_sender11111111")
	if f != nil {
		t.Error("friend should be removed after revoke")
	}
}

func TestManager_GetPendingRequests(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_ = mgr.SendRequest("mh_pending1", "P1", nil, "hi")
	_ = mgr.SendRequest("mh_pending2", "P2", nil, "hi")
	_, _ = mgr.AcceptRequest("mh_pending2", "Me", nil)

	pending := mgr.GetPendingRequests()
	if len(pending) != 1 {
		t.Errorf("expected 1 pending request, got %d", len(pending))
	}
	if pending[0].AgentID != "mh_pending1" {
		t.Errorf("expected mh_pending1, got %s", pending[0].AgentID)
	}
}

func TestManager_ListAcceptedFriends(t *testing.T) {
	store, _ := NewStore(testDBPath(t))
	defer store.Close()
	mgr := NewManager(store)

	_ = mgr.SendRequest("mh_friend1", "F1", nil, "hi")
	_ = mgr.SendRequest("mh_friend2", "F2", nil, "hi")
	_ = mgr.SendRequest("mh_friend3", "F3", nil, "hi")
	_, _ = mgr.AcceptRequest("mh_friend1", "Me", nil)
	_, _ = mgr.AcceptRequest("mh_friend2", "Me", nil)

	friends := mgr.ListAcceptedFriends()
	if len(friends) != 2 {
		t.Errorf("expected 2 accepted friends, got %d", len(friends))
	}
}
