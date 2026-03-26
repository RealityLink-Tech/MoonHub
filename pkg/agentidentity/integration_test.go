package agentidentity_test

import (
	"path/filepath"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
	"github.com/RealityLink-Tech/MoonHub/pkg/friends"
	"github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

// TestEndToEndFriendFlow tests the complete friend request lifecycle
// using all new packages together: identity, friends, and protocol.
func TestEndToEndFriendFlow(t *testing.T) {
	// Create two agent identities
	agentA, err := agentidentity.NewAgentIdentity("AgentA")
	if err != nil {
		t.Fatal(err)
	}
	agentB, err := agentidentity.NewAgentIdentity("AgentB")
	if err != nil {
		t.Fatal(err)
	}

	// Agent A sends friend request to Agent B
	env := mhp.NewEnvelope(agentA.AgentID, agentB.AgentID, mhp.MsgFriendRequest, mhp.FriendRequestPayload{
		AgentName: agentA.AgentName,
		Message:   "Let's collaborate!",
	})

	// Agent A signs the envelope
	sig, err := env.Sign(agentA.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}

	// Agent B verifies the signature
	if err := env.Verify(agentA.PublicKey); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}

	// Agent B decodes the payload
	payload, err := mhp.DecodePayload[mhp.FriendRequestPayload](env)
	if err != nil {
		t.Fatal(err)
	}
	if payload.AgentName != "AgentA" {
		t.Errorf("expected AgentA, got %s", payload.AgentName)
	}

	// Agent B stores the friend request
	dbPath := filepath.Join(t.TempDir(), "friends.db")
	store, err := friends.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mgr := friends.NewManager(store)
	if err := mgr.SendRequest(agentA.AgentID, payload.AgentName, agentA.PublicKeyBytes(), payload.Message); err != nil {
		t.Fatal(err)
	}

	// Agent B accepts the friend request
	acceptEnv := mhp.NewEnvelope(agentB.AgentID, agentA.AgentID, mhp.MsgFriendAccept, mhp.FriendAcceptPayload{
		AgentName: agentB.AgentName,
		PublicKey: agentB.PublicKeyBytes(),
	})
	acceptEnv.Sign(agentB.PrivateKey)

	// Agent A verifies the acceptance
	if err := acceptEnv.Verify(agentB.PublicKey); err != nil {
		t.Fatalf("accept signature verification failed: %v", err)
	}

	// Agent B's manager processes the acceptance
	friend, err := mgr.AcceptRequest(agentA.AgentID, agentB.AgentName, agentB.PublicKeyBytes())
	if err != nil {
		t.Fatal(err)
	}
	if friend.Status != friends.StatusAccepted {
		t.Errorf("expected accepted, got %s", friend.Status)
	}

	// Verify the friend is now in the accepted list
	accepted := mgr.ListAcceptedFriends()
	if len(accepted) != 1 {
		t.Fatalf("expected 1 accepted friend, got %d", len(accepted))
	}
	if accepted[0].AgentID != agentA.AgentID {
		t.Errorf("expected %s, got %s", agentA.AgentID, accepted[0].AgentID)
	}

	// Verify dedup rejects replayed message
	dedup := mhp.NewMessageDedup()
	if !dedup.CheckAndAdd(env.ID, env.Timestamp) {
		t.Error("first message should pass dedup")
	}
	if dedup.CheckAndAdd(env.ID, env.Timestamp) {
		t.Error("replayed message should fail dedup")
	}
}
