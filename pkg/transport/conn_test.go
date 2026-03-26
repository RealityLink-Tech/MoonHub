package transport

import (
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

func TestAgentConn_SendAndReceive(t *testing.T) {
	_, serverPriv, _ := ed25519.GenerateKey(nil)

	var received atomic.Pointer[mhp.Envelope]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		reader := mhp.NewEnvelopeReader(conn)
		env, err := reader.ReadEnvelope()
		if err != nil {
			return
		}
		received.Store(env)

		resp := mhp.NewEnvelope(env.To, env.From, mhp.MsgFriendPing, nil)
		_, _ = resp.Sign(serverPriv)
		writer, _ := mhp.NewEnvelopeWriter(conn)
		_ = writer.WriteEnvelope(resp)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	_, clientPriv, _ := ed25519.GenerateKey(nil)

	conn := NewAgentConn("mh_client000000000", "mh_server000000000", clientPriv, wsURL)
	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer conn.Close()

	env := mhp.NewEnvelope("mh_client000000000", "mh_server000000000", mhp.MsgFriendPing, nil)
	_, _ = env.Sign(clientPriv)

	if err := conn.Send(env); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	receivedEnv, err := conn.Receive()
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if receivedEnv.Type != mhp.MsgFriendPing {
		t.Errorf("expected type %s, got %s", mhp.MsgFriendPing, receivedEnv.Type)
	}
}

func TestAgentConn_Send_AfterClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	_, priv, _ := ed25519.GenerateKey(nil)
	conn := NewAgentConn("mh_client000000000", "mh_server000000000", priv, wsURL)
	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	conn.Close()

	env := mhp.NewEnvelope("mh_client000000000", "mh_server000000000", mhp.MsgFriendPing, nil)
	_, _ = env.Sign(priv)

	if err := conn.Send(env); err == nil {
		t.Error("expected error sending after close")
	}
}

func TestAgentConn_DoubleClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, _ := upgrader.Upgrade(w, r, nil)
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	_, priv, _ := ed25519.GenerateKey(nil)
	conn := NewAgentConn("mh_client000000000", "mh_server000000000", priv, wsURL)
	_ = conn.Connect()
	conn.Close()
	conn.Close()
}

func TestAgentConn_RemoteAgentID(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	conn := NewAgentConn("mh_client000000000", "mh_server000000000", priv, "ws://localhost:12345")
	if conn.RemoteAgentID() != "mh_server000000000" {
		t.Errorf("expected mh_server000000000, got %s", conn.RemoteAgentID())
	}
}

func TestAgentConn_StateTransitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, _ := upgrader.Upgrade(w, r, nil)
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	_, priv, _ := ed25519.GenerateKey(nil)
	conn := NewAgentConn("mh_client000000000", "mh_server000000000", priv, wsURL)

	if conn.State() != ConnDisconnected {
		t.Errorf("expected Disconnected, got %d", conn.State())
	}

	_ = conn.Connect()
	if conn.State() != ConnConnected {
		t.Errorf("expected Connected, got %d", conn.State())
	}

	conn.Close()
	if conn.State() != ConnDisconnected {
		t.Errorf("expected Disconnected after close, got %d", conn.State())
	}
}
