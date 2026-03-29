package moonhub

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	mhp "github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

// sendHandshake sends an initial envelope to complete the MoonHub WebSocket handshake.
func sendHandshake(t *testing.T, conn *websocket.Conn, from, to string, priv ed25519.PrivateKey) {
	t.Helper()
	env := mhp.NewEnvelope(from, to, mhp.MsgFriendPing, nil)
	_, _ = env.Sign(priv)
	writer, _ := mhp.NewEnvelopeWriter(conn)
	if err := writer.WriteEnvelope(env); err != nil {
		t.Fatalf("handshake write failed: %v", err)
	}
}

func TestMoonHubChannel_StartStop(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	ch, err := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Port:    0,
		Token:   "test-token",
	}, priv, msgBus)
	if err != nil {
		t.Fatal(err)
	}

	if ch.IsRunning() {
		t.Error("should not be running before Start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	err = ch.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !ch.IsRunning() {
		t.Error("should be running after Start")
	}

	ch.Stop(ctx)
	cancel()

	if ch.IsRunning() {
		t.Error("should not be running after Stop")
	}
}

func TestMoonHubChannel_WebSocketUpgrade(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	ch, _ := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Port:    0,
		Token:   "test-token",
	}, priv, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ch.Start(ctx)

	server := httptest.NewServer(ch)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	header := http.Header{}
	header.Set("Authorization", "Bearer test-token")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws", header)
	if err != nil {
		t.Fatalf("dial failed: %v (resp: %d)", err, resp.StatusCode)
	}
	resp.Body.Close()
	defer conn.Close()
}

func TestMoonHubChannel_Unauthorized(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	ch, _ := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Port:    0,
		Token:   "test-token",
	}, priv, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ch.Start(ctx)

	server := httptest.NewServer(ch)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
	if err == nil {
		conn.Close()
		resp.Body.Close()
		t.Fatal("expected error for unauthorized connection")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMoonHubChannel_ReceiveFriendRequest(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	var friendRequests []*mhp.FriendRequestPayload
	var mu sync.Mutex

	ch, _ := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Port:    0,
		Token:   "test-token",
	}, priv, msgBus)
	ch.SetOnFriendRequest(func(env *mhp.Envelope, payload *mhp.FriendRequestPayload) {
		mu.Lock()
		defer mu.Unlock()
		friendRequests = append(friendRequests, payload)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ch.Start(ctx)

	server := httptest.NewServer(ch)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	_, remotePriv, _ := ed25519.GenerateKey(nil)
	header := http.Header{}
	header.Set("Authorization", "Bearer test-token")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"/ws", header)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	defer conn.Close()

	env := mhp.NewEnvelope("mh_remote0000000000", ch.AgentID(), mhp.MsgFriendRequest, &mhp.FriendRequestPayload{
		AgentName: "RemoteAgent",
		Message:   "Let's be friends!",
	})
	_, _ = env.Sign(remotePriv)

	writer, _ := mhp.NewEnvelopeWriter(conn)
	_ = writer.WriteEnvelope(env)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(friendRequests) != 1 {
		t.Fatalf("expected 1 friend request, got %d", len(friendRequests))
	}
	if friendRequests[0].AgentName != "RemoteAgent" {
		t.Errorf("expected AgentName RemoteAgent, got %s", friendRequests[0].AgentName)
	}
}

func TestMoonHubChannel_ConnectionLimit(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	ch, _ := NewMoonHubChannel(config.MoonHubConfig{
		Enabled:        true,
		Port:           0,
		Token:          "test-token",
		MaxConnections: 1,
	}, priv, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ch.Start(ctx)

	server := httptest.NewServer(ch)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Set("Authorization", "Bearer test-token")

	conn1, resp1, err := websocket.DefaultDialer.Dial(wsURL+"/ws", header)
	if err != nil {
		t.Fatalf("first connection failed: %v", err)
	}
	resp1.Body.Close()
	defer conn1.Close()

	// Send handshake envelope to complete connection setup and increment connCount
	sendHandshake(t, conn1, "mh_remote0000000000", ch.AgentID(), priv)
	time.Sleep(50 * time.Millisecond)

	_, resp2, err := websocket.DefaultDialer.Dial(wsURL+"/ws", header)
	if err == nil {
		t.Fatal("expected second connection to fail")
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for connection limit, got %d", resp2.StatusCode)
	}
}

func TestMoonHubChannel_SendToAgent(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	msgBus := bus.NewMessageBus()

	ch, _ := NewMoonHubChannel(config.MoonHubConfig{
		Enabled: true,
		Port:    0,
		Token:   "test-token",
	}, priv, msgBus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ch.Start(ctx)

	server := httptest.NewServer(ch)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Set("Authorization", "Bearer test-token")

	conn, resp3, err := websocket.DefaultDialer.Dial(wsURL+"/ws", header)
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	defer conn.Close()

	// Send handshake envelope to complete connection setup
	sendHandshake(t, conn, "mh_remote0000000000", ch.AgentID(), priv)
	time.Sleep(50 * time.Millisecond)

	outMsg := bus.OutboundMessage{
		Channel: "moonhub",
		ChatID:  "moonhub:mh_remote0000000000",
		Content: "Hello from bus!",
	}

	if err := ch.Send(ctx, outMsg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var env mhp.Envelope
	if err := json.Unmarshal(rawMsg, &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if env.Type != mhp.MsgTaskResponse {
		t.Errorf("expected type %s, got %s", mhp.MsgTaskResponse, env.Type)
	}
}
