package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
	pkgapi "github.com/RealityLink-Tech/MoonHub/pkg/api"
	"github.com/google/uuid"
)

const maxChatBodyBytes = 1 << 20 // 1MB

// ChatHandler handles chat API requests (REST and SSE).
type ChatHandler struct {
	deviceStore *devices.DeviceStore
	agentLoop   *agent.AgentLoop
	chatHub     *pkgapi.ChatHub
}

// NewChatHandler creates a chat handler.
func NewChatHandler(deviceStore *devices.DeviceStore, chatHub *pkgapi.ChatHub) *ChatHandler {
	return &ChatHandler{
		deviceStore: deviceStore,
		chatHub:     chatHub,
	}
}

// SetAgentLoop updates the agent loop reference.
func (h *ChatHandler) SetAgentLoop(loop *agent.AgentLoop) {
	h.agentLoop = loop
}

// RegisterRoutes registers chat routes.
func (h *ChatHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/chat", h.authMiddleware(h.handleChat))
	mux.HandleFunc("POST /api/chat/stream", h.authMiddleware(h.handleChatStream))
	mux.HandleFunc("GET /api/chat/ws", h.handleChatWS)
}

// chatRequest is the JSON body for POST /api/chat and POST /api/chat/stream.
type chatRequest struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id,omitempty"`
}

// chatResponse is the JSON response for POST /api/chat.
type chatResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Content   string `json:"content"`
		SessionID string `json:"session_id"`
	} `json:"data,omitempty"`
	Error *chatError `json:"error,omitempty"`
}

type chatError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func writeChatError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(chatResponse{
		Success: false,
		Error:   &chatError{Message: message, Code: code},
	})
}

// authMiddleware validates Bearer token via DeviceStore.
func (h *ChatHandler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			writeChatError(w, http.StatusUnauthorized, "missing or invalid token", "TOKEN_INVALID")
			return
		}
		token := strings.TrimSpace(auth[7:])

		validation := h.deviceStore.ValidateToken(token)
		if !validation.Valid {
			writeChatError(w, http.StatusUnauthorized, "invalid or expired token", "TOKEN_INVALID")
			return
		}
		_ = h.deviceStore.UpdateLastSeen(validation.DeviceID, "")
		next.ServeHTTP(w, r)
	}
}

// handleChat handles POST /api/chat — synchronous chat.
func (h *ChatHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	if h.agentLoop == nil {
		writeChatError(w, http.StatusServiceUnavailable, "agent not available", "AGENT_UNAVAILABLE")
		return
	}

	var req chatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxChatBodyBytes)).Decode(&req); err != nil {
		writeChatError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeChatError(w, http.StatusBadRequest, "content is required", "INVALID_REQUEST")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	sessionKey := "lan:" + sessionID
	result, err := h.agentLoop.ProcessDirectWithChannel(r.Context(), req.Content, sessionKey, "lan", sessionID)
	if err != nil {
		writeChatError(w, http.StatusInternalServerError, "agent processing failed", "AGENT_ERROR")
		return
	}

	resp := chatResponse{Success: true}
	resp.Data.Content = result
	resp.Data.SessionID = sessionID

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleChatStream handles POST /api/chat/stream — SSE streaming.
func (h *ChatHandler) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if h.agentLoop == nil {
		writeChatError(w, http.StatusServiceUnavailable, "agent not available", "AGENT_UNAVAILABLE")
		return
	}

	var req chatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxChatBodyBytes)).Decode(&req); err != nil {
		writeChatError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		writeChatError(w, http.StatusBadRequest, "content is required", "INVALID_REQUEST")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	writeCORSHeaders(w, r)

	flusher, canFlush := w.(http.Flusher)

	// Send content_start
	fmt.Fprintf(w, "event: content_start\ndata: {\"session_id\":%q}\n\n", sessionID)
	if canFlush {
		flusher.Flush()
	}

	// Process (currently returns complete response; V2 will stream token-by-token)
	sessionKey := "lan:" + sessionID
	result, err := h.agentLoop.ProcessDirectWithChannel(r.Context(), req.Content, sessionKey, "lan", sessionID)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"message\":%q}\n\n", err.Error())
		if canFlush {
			flusher.Flush()
		}
		return
	}

	// Send content_chunk
	fmt.Fprintf(w, "event: content_chunk\ndata: {\"content\":%q,\"done\":false}\n\n", result)
	if canFlush {
		flusher.Flush()
	}

	// Send done
	fmt.Fprintf(w, "event: done\ndata: {\"content\":%q,\"done\":true}\n\n", result)
	if canFlush {
		flusher.Flush()
	}
}

// handleChatWS handles GET /api/chat/ws — WebSocket chat.
// Delegates to pkg/api ChatHub which handles the WebSocket lifecycle.
func (h *ChatHandler) handleChatWS(w http.ResponseWriter, r *http.Request) {
	// Validate token from query param
	token := r.URL.Query().Get("token")
	if token == "" {
		pkgapi.WriteJSONError(w, http.StatusUnauthorized, "missing token")
		return
	}

	validation := h.deviceStore.ValidateToken(token)
	if !validation.Valid {
		pkgapi.WriteJSONError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if h.chatHub == nil {
		http.Error(w, "chat not available", http.StatusServiceUnavailable)
		return
	}

	// Delegate to the shared WebSocket handler from pkg/api
	h.chatHub.HandleWebSocket(w, r)
}
