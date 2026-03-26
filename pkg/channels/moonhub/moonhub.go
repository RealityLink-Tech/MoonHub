package moonhub

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
	mhp "github.com/RealityLink-Tech/MoonHub/pkg/protocol/mhp"
)

const moonhubPathPrefix = "/agent"

// agentConn wraps a WebSocket connection with metadata.
type agentConn struct {
	id      string
	conn    *websocket.Conn
	agentID string
	writeMu sync.Mutex
	closed  atomic.Bool
}

func (ac *agentConn) writeJSON(v any) error {
	if ac.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	ac.writeMu.Lock()
	defer ac.writeMu.Unlock()
	return ac.conn.WriteJSON(v)
}

func (ac *agentConn) close() {
	if ac.closed.CompareAndSwap(false, true) {
		ac.conn.Close()
	}
}

// noopFriendStore accepts all friend requests (for initial handshake).
type noopFriendStore struct{}

func (s *noopFriendStore) IsFriend(agentID string) bool        { return false }
func (s *noopFriendStore) GetPublicKey(agentID string) []byte { return nil }

// MoonHubChannel implements the agent-to-agent channel via WebSocket.
type MoonHubChannel struct {
	*channels.BaseChannel
	config      config.MoonHubConfig
	agentID     string
	privKey     ed25519.PrivateKey
	upgrader    websocket.Upgrader
	connections sync.Map // connID → *agentConn
	agentConns  sync.Map // agentID → *agentConn (latest connection per agent)
	connCount   atomic.Int32
	handler     *mhp.Handler
	onFriendReq mhp.FriendRequestCallback
	onFriendAcc mhp.FriendAcceptCallback
	onFriendRej mhp.FriendRejectCallback
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewMoonHubChannel creates a new MoonHub agent-to-agent channel.
func NewMoonHubChannel(cfg config.MoonHubConfig, privKey ed25519.PrivateKey, messageBus *bus.MessageBus) (*MoonHubChannel, error) {
	var agentID string
	if privKey != nil {
		pub := privKey.Public().(ed25519.PublicKey)
		agentID = agentidentity.DeriveAgentID(pub)
	}

	base := channels.NewBaseChannel("moonhub", cfg, messageBus, cfg.AllowFrom)

	ch := &MoonHubChannel{
		BaseChannel: base,
		config:      cfg,
		agentID:     agentID,
		privKey:     privKey,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
		},
	}

	ch.handler = mhp.NewHandler(mhp.HandlerConfig{
		AgentID:     agentID,
		FriendStore: &noopFriendStore{},
		OnFriendRequest: func(env *mhp.Envelope, payload *mhp.FriendRequestPayload) {
			if ch.onFriendReq != nil {
				ch.onFriendReq(env, payload)
			}
		},
		OnFriendAccept: func(env *mhp.Envelope, payload *mhp.FriendAcceptPayload) {
			if ch.onFriendAcc != nil {
				ch.onFriendAcc(env, payload)
			}
		},
		OnFriendReject: func(env *mhp.Envelope, payload *mhp.FriendRejectPayload) {
			if ch.onFriendRej != nil {
				ch.onFriendRej(env, payload)
			}
		},
	})

	return ch, nil
}

// AgentID returns this agent's derived ID.
func (c *MoonHubChannel) AgentID() string { return c.agentID }

// SetOnFriendRequest sets the callback for incoming friend requests.
func (c *MoonHubChannel) SetOnFriendRequest(cb mhp.FriendRequestCallback) {
	c.onFriendReq = cb
}

// SetOnFriendAccept sets the callback for incoming friend acceptances.
func (c *MoonHubChannel) SetOnFriendAccept(cb mhp.FriendAcceptCallback) {
	c.onFriendAcc = cb
}

// SetPrivateKey sets or updates the agent's private key.
func (c *MoonHubChannel) SetPrivateKey(privKey ed25519.PrivateKey) {
	c.privKey = privKey
	pub := privKey.Public().(ed25519.PublicKey)
	c.agentID = agentidentity.DeriveAgentID(pub)
}

func (c *MoonHubChannel) Start(ctx context.Context) error {
	logger.InfoC("moonhub", "Starting MoonHub agent channel")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	logger.InfoCF("moonhub", "MoonHub agent channel started", map[string]any{
		"agent_id": c.agentID,
	})
	return nil
}

func (c *MoonHubChannel) Stop(ctx context.Context) error {
	logger.InfoC("moonhub", "Stopping MoonHub agent channel")
	c.SetRunning(false)

	c.connections.Range(func(key, value any) bool {
		if ac, ok := value.(*agentConn); ok {
			ac.close()
		}
		c.connections.Delete(key)
		return true
	})
	c.agentConns.Clear()

	if c.cancel != nil {
		c.cancel()
	}

	logger.InfoC("moonhub", "MoonHub agent channel stopped")
	return nil
}

func (c *MoonHubChannel) WebhookPath() string { return moonhubPathPrefix + "/" }

func (c *MoonHubChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, moonhubPathPrefix)

	switch {
	case path == "/ws" || path == "/ws/":
		c.handleWebSocket(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Send implements Channel — sends a message to a connected agent.
// ChatID format: "moonhub:<agentID>"
func (c *MoonHubChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	agentID := strings.TrimPrefix(msg.ChatID, "moonhub:")

	acVal, ok := c.agentConns.Load(agentID)
	if !ok {
		return fmt.Errorf("no active connection for agent %s: %w", agentID, channels.ErrSendFailed)
	}
	ac := acVal.(*agentConn)

	outEnv := mhp.NewEnvelope(c.agentID, agentID, mhp.MsgTaskResponse, map[string]any{
		"content": msg.Content,
	})
	_, _ = outEnv.Sign(c.privKey)

	return ac.writeJSON(outEnv)
}

func (c *MoonHubChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !c.IsRunning() {
		http.Error(w, "channel not running", http.StatusServiceUnavailable)
		return
	}

	if !c.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	maxConns := c.config.MaxConnections
	if maxConns <= 0 {
		maxConns = 50
	}
	if int(c.connCount.Load()) >= maxConns {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCF("moonhub", "WebSocket upgrade failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	reader := mhp.NewEnvelopeReader(conn)
	firstEnv, err := reader.ReadEnvelope()
	if err != nil {
		conn.Close()
		return
	}

	remoteAgentID := firstEnv.From

	ac := &agentConn{
		id:      uuid.New().String(),
		conn:    conn,
		agentID: remoteAgentID,
	}

	c.connections.Store(ac.id, ac)
	c.agentConns.Store(remoteAgentID, ac)
	c.connCount.Add(1)

	logger.InfoCF("moonhub", "Agent connected", map[string]any{
		"agent_id": remoteAgentID,
		"conn_id":  ac.id,
	})

	c.handler.HandleEnvelope(firstEnv)
	go c.readLoop(ac, reader)
}

func (c *MoonHubChannel) authenticate(r *http.Request) bool {
	if c.config.Token == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return after == c.config.Token
	}
	return false
}

func (c *MoonHubChannel) readLoop(ac *agentConn, reader *mhp.EnvelopeReader) {
	defer func() {
		ac.close()
		c.connections.Delete(ac.id)
		c.connCount.Add(-1)
		if current, ok := c.agentConns.Load(ac.agentID); ok && current.(*agentConn) == ac {
			c.agentConns.Delete(ac.agentID)
		}
		logger.InfoCF("moonhub", "Agent disconnected", map[string]any{
			"agent_id": ac.agentID,
			"conn_id":  ac.id,
		})
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		env, err := reader.ReadEnvelope()
		if err != nil {
			return
		}

		c.handler.HandleEnvelope(env)
	}
}
