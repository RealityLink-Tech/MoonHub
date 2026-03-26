package mhp

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWriteEnvelope_SendsValidJSON(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}

		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if env.Version != ProtocolVersion {
			t.Errorf("expected version %d, got %d", ProtocolVersion, env.Version)
		}
		if env.From != "mh_abcdef1234567890" {
			t.Errorf("expected from mh_abcdef1234567890, got %s", env.From)
		}
		if env.To != "mh_1111111111111111" {
			t.Errorf("expected to mh_1111111111111111, got %s", env.To)
		}
		if env.Type != MsgFriendRequest {
			t.Errorf("expected type %s, got %s", MsgFriendRequest, env.Type)
		}
		if env.Signature == "" {
			t.Error("expected non-empty signature")
		}

		// Verify signature
		if err := env.Verify(pub); err != nil {
			t.Errorf("signature verification failed: %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	env := NewEnvelope("mh_abcdef1234567890", "mh_1111111111111111", MsgFriendRequest, &FriendRequestPayload{
		AgentName: "TestAgent",
		Message:   "Hello!",
	})
	_, err = env.Sign(priv)
	if err != nil {
		t.Fatal(err)
	}

	writer, err := NewEnvelopeWriter(conn)
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.WriteEnvelope(env); err != nil {
		t.Fatalf("WriteEnvelope failed: %v", err)
	}
}

func TestEnvelopeReader_ReadValidEnvelope(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		env := NewEnvelope("mh_server000000000", "mh_client000000000", MsgFriendAccept, &FriendAcceptPayload{
			AgentName: "ServerAgent",
			PublicKey: pub,
		})
		_, _ = env.Sign(priv)

		writer, _ := NewEnvelopeWriter(conn)
		_ = writer.WriteEnvelope(env)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reader := NewEnvelopeReader(conn)

	env, err := reader.ReadEnvelope()
	if err != nil {
		t.Fatalf("ReadEnvelope failed: %v", err)
	}

	if env.Type != MsgFriendAccept {
		t.Errorf("expected type %s, got %s", MsgFriendAccept, env.Type)
	}
	if env.Signature == "" {
		t.Error("expected non-empty signature")
	}
}

func TestEnvelopeReader_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.TextMessage, []byte("not json"))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reader := NewEnvelopeReader(conn)
	_, err = reader.ReadEnvelope()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestEnvelopeReader_ConnectionClosed(t *testing.T) {
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
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reader := NewEnvelopeReader(conn)

	_, err = reader.ReadEnvelope()
	if err == nil {
		t.Error("expected error for closed connection")
	}
}

func TestWriteEnvelope_WriteLock(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		for i := 0; i < 3; i++ {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	writer, err := NewEnvelopeWriter(conn)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	for i := 0; i < 3; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			env := NewEnvelope("mh_abcdef1234567890", "mh_1111111111111111", MsgFriendPing, nil)
			_, _ = env.Sign(priv)
			_ = writer.WriteEnvelope(env)
		}()
	}

	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("concurrent writes timed out — possible deadlock")
		}
	}
}
