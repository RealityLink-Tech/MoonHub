package mhp

import (
	"crypto/ed25519"
	"log"
	"time"
)

// HandlerResult indicates how an envelope was handled.
type HandlerResult int

const (
	HandlerAccepted  HandlerResult = iota
	HandlerRejected
	HandlerDeduped
)

// HandlerFriendStore provides friend lookup for the mhp handler.
type HandlerFriendStore interface {
	IsFriend(agentID string) bool
	GetPublicKey(agentID string) []byte
}

// FriendRequestCallback is called when a friend_request envelope is received from a new agent.
type FriendRequestCallback func(env *Envelope, payload *FriendRequestPayload)

// FriendAcceptCallback is called when a friend_accept envelope is received.
type FriendAcceptCallback func(env *Envelope, payload *FriendAcceptPayload)

// FriendRejectCallback is called when a friend_reject envelope is received.
type FriendRejectCallback func(env *Envelope, payload *FriendRejectPayload)

// HandlerConfig holds configuration for the mhp handler.
type HandlerConfig struct {
	AgentID         string
	FriendStore     HandlerFriendStore
	OnFriendRequest FriendRequestCallback
	OnFriendAccept  FriendAcceptCallback
	OnFriendReject  FriendRejectCallback
}

// Handler processes inbound mhp Envelopes.
type Handler struct {
	config HandlerConfig
	dedup  *MessageDedup
}

// NewHandler creates a new mhp handler.
func NewHandler(config HandlerConfig) *Handler {
	return &Handler{
		config: config,
		dedup:  NewMessageDedup(),
	}
}

// HandleEnvelope processes an inbound envelope and returns a HandlerResult.
func (h *Handler) HandleEnvelope(env *Envelope) HandlerResult {
	// Check for stale envelope first.
	if time.Since(time.Unix(env.Timestamp, 0)) > MaxMessageAge {
		return HandlerRejected
	}

	// Check for duplicate.
	if !h.dedup.CheckAndAdd(env.ID, env.Timestamp) {
		return HandlerDeduped
	}

	if env.To != h.config.AgentID {
		log.Printf("mhp: envelope addressed to %s, not us (%s)", env.To, h.config.AgentID)
		return HandlerRejected
	}

	pubKey := h.config.FriendStore.GetPublicKey(env.From)

	if pubKey == nil && env.Type != MsgFriendRequest {
		log.Printf("mhp: unknown sender %s for type %s", env.From, env.Type)
		return HandlerRejected
	}

	if pubKey != nil {
		if err := env.Verify(ed25519.PublicKey(pubKey)); err != nil {
			log.Printf("mhp: signature verification failed from %s: %v", env.From, err)
			return HandlerRejected
		}
	}

	switch env.Type {
	case MsgFriendRequest:
		return h.handleFriendRequest(env)
	case MsgFriendAccept:
		return h.handleFriendAccept(env)
	case MsgFriendReject:
		return h.handleFriendReject(env)
	default:
		log.Printf("mhp: unhandled message type %s from %s", env.Type, env.From)
		return HandlerRejected
	}
}

func (h *Handler) handleFriendRequest(env *Envelope) HandlerResult {
	if h.config.OnFriendRequest == nil {
		return HandlerRejected
	}

	payload, err := DecodePayload[FriendRequestPayload](env)
	if err != nil {
		log.Printf("mhp: failed to decode friend_request payload: %v", err)
		return HandlerRejected
	}

	if h.config.FriendStore.GetPublicKey(env.From) != nil {
		log.Printf("mhp: friend request already exists from %s", env.From)
		return HandlerRejected
	}

	h.config.OnFriendRequest(env, payload)
	return HandlerAccepted
}

func (h *Handler) handleFriendAccept(env *Envelope) HandlerResult {
	payload, err := DecodePayload[FriendAcceptPayload](env)
	if err != nil {
		log.Printf("mhp: failed to decode friend_accept payload: %v", err)
		return HandlerRejected
	}

	if h.config.OnFriendAccept != nil {
		h.config.OnFriendAccept(env, payload)
	}
	return HandlerAccepted
}

func (h *Handler) handleFriendReject(env *Envelope) HandlerResult {
	payload, err := DecodePayload[FriendRejectPayload](env)
	if err != nil {
		log.Printf("mhp: failed to decode friend_reject payload: %v", err)
		return HandlerRejected
	}

	if h.config.OnFriendReject != nil {
		h.config.OnFriendReject(env, payload)
	}
	return HandlerRejected
}
