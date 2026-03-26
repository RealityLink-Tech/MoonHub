// cloud/directory/handler.go
package directory

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
)

type RegisterRequest struct {
	AgentID   string `json:"agentID"`
	AgentName string `json:"agentName"`
	PublicKey string `json:"publicKey"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

type HeartbeatRequest struct {
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

type DeleteRequest struct {
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

type AgentResponse struct {
	AgentID              string `json:"agentID"`
	AgentName            string `json:"agentName"`
	Online               bool   `json:"online"`
	PublicKeyFingerprint string `json:"publicKeyFingerprint"`
	RelayEndpoint        string `json:"relayEndpoint,omitempty"`
}

type PublicKeyResponse struct {
	AgentID   string `json:"agentID"`
	PublicKey string `json:"publicKey"`
}

type Handler struct {
	store AgentStore
	cache OnlineCache
}

func NewHandler(store AgentStore, cache OnlineCache) *Handler {
	return &Handler{store: store, cache: cache}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/agents/register":
		h.handleRegister(w, r)
	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/heartbeat"):
		agentID := strings.TrimSuffix(r.URL.Path, "/heartbeat")
		agentID = strings.TrimPrefix(agentID, "/agents/")
		h.handleHeartbeat(w, r, agentID)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pubkey"):
		agentID := strings.TrimSuffix(r.URL.Path, "/pubkey")
		agentID = strings.TrimPrefix(agentID, "/agents/")
		h.handleGetPublicKey(w, r, agentID)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/agents/"):
		agentID := strings.TrimPrefix(r.URL.Path, "/agents/")
		h.handleGetAgent(w, r, agentID)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/agents/"):
		agentID := strings.TrimPrefix(r.URL.Path, "/agents/")
		h.handleDelete(w, r, agentID)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		http.Error(w, "invalid publicKey encoding", http.StatusBadRequest)
		return
	}

	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		http.Error(w, "invalid signature encoding", http.StatusBadRequest)
		return
	}

	// Verify signature: use stored key for re-registration (prevents key replacement)
	verifyKey := pubKeyBytes
	if storedPubKey, err := h.store.GetPublicKey(r.Context(), req.AgentID); err == nil {
		verifyKey = storedPubKey
	}
	msg := fmt.Sprintf("%s%s%s%d", req.AgentID, req.AgentName, req.PublicKey, req.Timestamp)
	if !ed25519.Verify(verifyKey, []byte(msg), sigBytes) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Verify agentID matches public key
	expectedID := agentidentity.DeriveAgentID(pubKeyBytes)
	if req.AgentID != expectedID {
		http.Error(w, "agentID does not match public key", http.StatusBadRequest)
		return
	}

	// Check timestamp freshness (5 minute window)
	if time.Since(time.Unix(req.Timestamp, 0)) > 5*time.Minute {
		http.Error(w, "timestamp too old", http.StatusBadRequest)
		return
	}

	record := &AgentRecord{
		AgentID:    req.AgentID,
		AgentName:  req.AgentName,
		PublicKey:  pubKeyBytes,
		Endpoint:   req.Endpoint,
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
	}

	if err := h.store.UpsertAgent(r.Context(), record); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.cache.MarkOnline(r.Context(), req.AgentID, req.Endpoint)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleHeartbeat(w http.ResponseWriter, r *http.Request, agentID string) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if time.Since(time.Unix(req.Timestamp, 0)) > 5*time.Minute {
		http.Error(w, "timestamp too old", http.StatusBadRequest)
		return
	}

	pubKey, err := h.store.GetPublicKey(r.Context(), agentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		http.Error(w, "invalid signature encoding", http.StatusBadRequest)
		return
	}
	msg := fmt.Sprintf("%s%d", agentID, req.Timestamp)
	if !ed25519.Verify(pubKey, []byte(msg), sigBytes) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	agent, err := h.store.GetAgent(r.Context(), agentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	agent.LastSeenAt = time.Now()
	h.store.UpsertAgent(r.Context(), agent)
	h.cache.MarkOnline(r.Context(), agentID, agent.Endpoint)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleGetAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	agent, err := h.store.GetAgent(r.Context(), agentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	online, endpoint, _ := h.cache.IsOnline(r.Context(), agentID)
	fpHash := sha256.Sum256(agent.PublicKey)
	fingerprint := "sha256:" + hex.EncodeToString(fpHash[:])[:8]

	resp := AgentResponse{
		AgentID:              agent.AgentID,
		AgentName:            agent.AgentName,
		Online:               online,
		PublicKeyFingerprint: fingerprint,
		RelayEndpoint:        endpoint,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleGetPublicKey(w http.ResponseWriter, r *http.Request, agentID string) {
	pubKey, err := h.store.GetPublicKey(r.Context(), agentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	resp := PublicKeyResponse{
		AgentID:   agentID,
		PublicKey: base64.StdEncoding.EncodeToString(pubKey),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request, agentID string) {
	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if time.Since(time.Unix(req.Timestamp, 0)) > 5*time.Minute {
		http.Error(w, "timestamp too old", http.StatusBadRequest)
		return
	}

	pubKey, err := h.store.GetPublicKey(r.Context(), agentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		http.Error(w, "invalid signature encoding", http.StatusBadRequest)
		return
	}
	msg := fmt.Sprintf("%s%d", agentID, req.Timestamp)
	if !ed25519.Verify(pubKey, []byte(msg), sigBytes) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	h.store.DeleteAgent(r.Context(), agentID)
	h.cache.MarkOffline(r.Context(), agentID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
