package moonhub

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/friends"
	mhp "github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

func TestIntegration_FriendRequestOverWebSocket(t *testing.T) {
	_, alicePriv, _ := ed25519.GenerateKey(nil)
	_, bobPriv, _ := ed25519.GenerateKey(nil)

	alicePub := alicePriv.Public().(ed25519.PublicKey)
	aliceID := agentidentity.DeriveAgentID(alicePub)

	bobBus := bus.NewMessageBus()
	bobStore, err := friends.NewStore(t.TempDir() + "/friends.db")
	if err != nil {
		t.Fatal(err)
	}
	defer bobStore.Close()
	bobFriendMgr := friends.NewManager(bobStore)

	var receivedRequest *mhp.FriendRequestPayload
	var mu sync.Mutex

	bobCh, err := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Token:   "test-token",
	}, bobPriv, bobBus)
	if err != nil {
		t.Fatal(err)
	}

	bobCh.SetOnFriendRequest(func(env *mhp.Envelope, payload *mhp.FriendRequestPayload) {
		mu.Lock()
		defer mu.Unlock()
		receivedRequest = payload

		_ = bobFriendMgr.SendRequest(env.From, payload.AgentName, nil, payload.Message)
		_, _ = bobFriendMgr.AcceptRequest(env.From, "Bob", bobPriv.Public().(ed25519.PublicKey))
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = bobCh.Start(ctx)

	bobServer := httptest.NewServer(bobCh)
	defer bobServer.Close()

	wsURL := "ws" + strings.TrimPrefix(bobServer.URL, "http")
	header := http.Header{}
	header.Set("Authorization", "Bearer test-token")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/ws", header)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	env := mhp.NewEnvelope(aliceID, bobCh.AgentID(), mhp.MsgFriendRequest, &mhp.FriendRequestPayload{
		AgentName: "Alice",
		Message:   "Hi Bob!",
	})
	_, _ = env.Sign(alicePriv)

	writer, _ := mhp.NewEnvelopeWriter(conn)
	_ = writer.WriteEnvelope(env)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if receivedRequest == nil {
		t.Fatal("Bob did not receive friend request")
	}
	if receivedRequest.AgentName != "Alice" {
		t.Errorf("expected AgentName Alice, got %s", receivedRequest.AgentName)
	}
	if receivedRequest.Message != "Hi Bob!" {
		t.Errorf("expected Message 'Hi Bob!', got %s", receivedRequest.Message)
	}

	accepted := bobFriendMgr.ListAcceptedFriends()
	if len(accepted) == 0 {
		t.Error("expected Alice to be in accepted friends")
	}
	if accepted[0].AgentID != aliceID {
		t.Errorf("expected friend ID %s, got %s", aliceID, accepted[0].AgentID)
	}
}

func TestIntegration_UnauthorizedConnection(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	ch, err := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Token:   "secret-token",
	}, priv, msgBus)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ch.Start(ctx)

	server := httptest.NewServer(ch)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
	if err == nil {
		conn.Close()
		t.Fatal("expected error with wrong token")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}
