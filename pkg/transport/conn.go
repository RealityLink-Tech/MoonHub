package transport

import (
	"crypto/ed25519"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

type ConnState int32

const (
	ConnDisconnected ConnState = iota
	ConnConnected
	ConnConnecting
)

type AgentConn struct {
	localID   string
	remoteID  string
	privKey   ed25519.PrivateKey
	wsURL     string
	conn      *websocket.Conn
	writer    *mhp.EnvelopeWriter
	reader    *mhp.EnvelopeReader
	state     atomic.Int32
	closeOnce sync.Once
}

func NewAgentConn(localID, remoteID string, privKey ed25519.PrivateKey, wsURL string) *AgentConn {
	c := &AgentConn{
		localID:  localID,
		remoteID: remoteID,
		privKey:  privKey,
		wsURL:    wsURL,
	}
	c.state.Store(int32(ConnDisconnected))
	return c
}

func (c *AgentConn) RemoteAgentID() string { return c.remoteID }

func (c *AgentConn) State() ConnState { return ConnState(c.state.Load()) }

func (c *AgentConn) Connect() error {
	c.state.Store(int32(ConnConnecting))

	conn, _, err := websocket.DefaultDialer.Dial(c.wsURL, nil)
	if err != nil {
		c.state.Store(int32(ConnDisconnected))
		return fmt.Errorf("dial %s: %w", c.wsURL, err)
	}

	c.conn = conn
	writer, err := mhp.NewEnvelopeWriter(conn)
	if err != nil {
		conn.Close()
		c.state.Store(int32(ConnDisconnected))
		return fmt.Errorf("create writer: %w", err)
	}
	c.writer = writer
	c.reader = mhp.NewEnvelopeReader(conn)
	c.state.Store(int32(ConnConnected))
	return nil
}

func (c *AgentConn) Send(env *mhp.Envelope) error {
	if c.State() != ConnConnected {
		return fmt.Errorf("connection not connected")
	}
	return c.writer.WriteEnvelope(env)
}

func (c *AgentConn) Receive() (*mhp.Envelope, error) {
	if c.State() != ConnConnected {
		return nil, fmt.Errorf("connection not connected")
	}
	return c.reader.ReadEnvelope()
}

func (c *AgentConn) Close() {
	c.closeOnce.Do(func() {
		c.state.Store(int32(ConnDisconnected))
		if c.conn != nil {
			c.conn.Close()
		}
	})
}
