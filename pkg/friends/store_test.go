package friends

import (
	"os"
	"path/filepath"
	"testing"
)

func testDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "friends_test.db")
}

func TestNewStore(t *testing.T) {
	path := testDB(t)
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestStore_AddAndGetFriend(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	friend := &Friend{
		AgentID:    "mh_aaaaaaaa11111111",
		AgentName:  "TestAgent",
		PublicKey:  []byte("fake-public-key-32bytes!!"),
		Status:     StatusAccepted,
		AddedAt:    store.now(),
	}

	if err := store.AddFriend(friend); err != nil {
		t.Fatal(err)
	}

	got := store.GetFriend("mh_aaaaaaaa11111111")
	if got == nil {
		t.Fatal("expected friend, got nil")
	}
	if got.AgentName != "TestAgent" {
		t.Errorf("expected name TestAgent, got %s", got.AgentName)
	}
	if got.Status != StatusAccepted {
		t.Errorf("expected status accepted, got %s", got.Status)
	}
}

func TestStore_GetFriend_NotFound(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	got := store.GetFriend("mh_nonexistent")
	if got != nil {
		t.Error("expected nil for nonexistent friend")
	}
}

func TestStore_RemoveFriend(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	friend := &Friend{
		AgentID:   "mh_aaaaaaaa11111111",
		AgentName: "ToRemove",
		Status:    StatusAccepted,
		AddedAt:   store.now(),
	}
	store.AddFriend(friend)

	if err := store.RemoveFriend("mh_aaaaaaaa11111111"); err != nil {
		t.Fatal(err)
	}

	if store.GetFriend("mh_aaaaaaaa11111111") != nil {
		t.Error("friend should be removed")
	}
}

func TestStore_ListFriends(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	store.AddFriend(&Friend{AgentID: "mh_aaaa1", AgentName: "Agent1", Status: StatusAccepted, AddedAt: store.now()})
	store.AddFriend(&Friend{AgentID: "mh_aaaa2", AgentName: "Agent2", Status: StatusAccepted, AddedAt: store.now()})
	store.AddFriend(&Friend{AgentID: "mh_aaaa3", AgentName: "Agent3", Status: StatusPending, AddedAt: store.now()})

	friends := store.ListFriends(StatusAccepted)
	if len(friends) != 2 {
		t.Errorf("expected 2 accepted friends, got %d", len(friends))
	}

	all := store.ListFriends(StatusAny)
	if len(all) != 3 {
		t.Errorf("expected 3 total friends, got %d", len(all))
	}
}

func TestStore_UpdateStatus(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	store.AddFriend(&Friend{AgentID: "mh_aaaa1", AgentName: "Pending", Status: StatusPending, AddedAt: store.now()})

	if err := store.UpdateStatus("mh_aaaa1", StatusAccepted); err != nil {
		t.Fatal(err)
	}

	friend := store.GetFriend("mh_aaaa1")
	if friend.Status != StatusAccepted {
		t.Errorf("expected accepted, got %s", friend.Status)
	}
}

func TestStore_UpdateStatus_NotFound(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	err := store.UpdateStatus("mh_nonexistent", StatusAccepted)
	if err == nil {
		t.Error("expected error for nonexistent friend")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestStore_IsFriend(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	defer store.Close()

	store.AddFriend(&Friend{AgentID: "mh_aaaa1", Status: StatusAccepted, AddedAt: store.now()})

	if !store.IsFriend("mh_aaaa1") {
		t.Error("expected IsFriend true for accepted friend")
	}
	if store.IsFriend("mh_nonexistent") {
		t.Error("expected IsFriend false for nonexistent")
	}
}

func TestStore_PersistAcrossReopen(t *testing.T) {
	path := testDB(t)

	store1, _ := NewStore(path)
	store1.AddFriend(&Friend{AgentID: "mh_persist", AgentName: "Persistent", Status: StatusAccepted, AddedAt: store1.now()})
	store1.Close()

	store2, _ := NewStore(path)
	defer store2.Close()

	friend := store2.GetFriend("mh_persist")
	if friend == nil {
		t.Fatal("friend should persist across reopen")
	}
	if friend.AgentName != "Persistent" {
		t.Errorf("expected Persistent, got %s", friend.AgentName)
	}
}

func TestStore_Close(t *testing.T) {
	path := testDB(t)
	store, _ := NewStore(path)
	store.Close()

	// Double close should not panic
	store.Close()
}
