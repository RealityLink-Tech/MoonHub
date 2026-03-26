// cloud/relay/bridge_test.go
package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestBridge_ConnectAndBridge(t *testing.T) {
	bridge := NewBridge()
	go bridge.Run()
	defer bridge.Stop()

	// Track when both agents are connected
	agentAConnected := make(chan struct{})
	agentBConnected := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			return
		}

		agentID := r.URL.Query().Get("agent_id")
		if agentID == "mh_aaaa1111bbbb2222" {
			close(agentAConnected)
		} else if agentID == "mh_cccc3333dddd4444" {
			close(agentBConnected)
		}

		bridge.RegisterAgent(agentID, conn)
		bridge.HandleConnection(agentID, conn)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect agent A
	wsURLA := wsURL + "?agent_id=mh_aaaa1111bbbb2222"
	dialedConnA, respA, err := websocket.DefaultDialer.Dial(wsURLA, nil)
	if err != nil {
		t.Fatal(err)
	}
	respA.Body.Close()
	defer dialedConnA.Close()

	// Wait for agent A to be registered
	<-agentAConnected

	// Connect agent B
	wsURLB := wsURL + "?agent_id=mh_cccc3333dddd4444"
	dialedConnB, respB, err := websocket.DefaultDialer.Dial(wsURLB, nil)
	if err != nil {
		t.Fatal(err)
	}
	respB.Body.Close()
	defer dialedConnB.Close()

	// Wait for agent B to be registered
	<-agentBConnected

	// A requests bridge to B
	err = dialedConnA.WriteMessage(websocket.TextMessage, []byte("CONNECT mh_cccc3333dddd4444"))
	if err != nil {
		t.Fatal(err)
	}

	// B should receive CONNECTED
	_, msg, err := dialedConnB.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "CONNECTED" {
		t.Errorf("B got %q, want CONNECTED", string(msg))
	}

	// A should receive CONNECTED
	_, msg, err = dialedConnA.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "CONNECTED" {
		t.Errorf("A got %q, want CONNECTED", string(msg))
	}
}

func TestBridge_TargetOffline(t *testing.T) {
	bridge := NewBridge()
	go bridge.Run()
	defer bridge.Stop()

	agentAConnected := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			return
		}

		agentID := r.URL.Query().Get("agent_id")
		if agentID == "mh_aaaa1111bbbb2222" {
			close(agentAConnected)
		}

		bridge.RegisterAgent(agentID, conn)
		bridge.HandleConnection(agentID, conn)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURLA := wsURL + "?agent_id=mh_aaaa1111bbbb2222"
	connA, respA2, err := websocket.DefaultDialer.Dial(wsURLA, nil)
	if err != nil {
		t.Fatal(err)
	}
	respA2.Body.Close()
	defer connA.Close()

	// Wait for agent A to be registered
	<-agentAConnected

	err = connA.WriteMessage(websocket.TextMessage, []byte("CONNECT mh_nonexistent"))
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := connA.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "ERROR target_offline" {
		t.Errorf("got %q, want ERROR target_offline", string(msg))
	}
}
