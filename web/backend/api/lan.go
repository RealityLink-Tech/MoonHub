package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
	"github.com/RealityLink-Tech/MoonHub/pkg/social"
)

// LANHandler handles LAN API requests for chat functionality.
type LANHandler struct {
	deviceStore *devices.DeviceStore
	agentLoop   *agent.AgentLoop
	sessions    *social.SessionManager
	deviceID    string
	deviceName  string
	version     string
}

// NewLANHandler creates a new LAN API handler.
func NewLANHandler(
	deviceStore *devices.DeviceStore,
	agentLoop *agent.AgentLoop,
	deviceID, deviceName, version string,
) *LANHandler {
	return &LANHandler{
		deviceStore: deviceStore,
		agentLoop:   agentLoop,
		sessions:    social.NewSessionManager(),
		deviceID:    deviceID,
		deviceName:  deviceName,
		version:     version,
	}
}

// RegisterRoutes registers LAN API routes with the mux.
func (h *LANHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/lan/info", h.authMiddleware(h.handleInfo))
	mux.HandleFunc("POST /api/lan/chat", h.authMiddleware(h.handleChat))
	mux.HandleFunc("GET /api/lan/chat/stream", h.authMiddleware(h.handleChatStream))
}

// authContextKey is the context key for auth info.
type authContextKey string

const authDeviceIDKey authContextKey = "deviceId"

// authMiddleware validates Bearer token and injects device ID into context.
func (h *LANHandler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireLANClientLANAPI(w, r) {
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeLANError(w, http.StatusUnauthorized, "authorization header required")
			return
		}

		// Check Bearer format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeLANError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		token := parts[1]

		// Validate token
		validation := h.deviceStore.ValidateToken(token)
		if !validation.Valid {
			writeLANError(w, http.StatusUnauthorized, validation.Error)
			return
		}

		// Update last seen
		_ = h.deviceStore.UpdateLastSeen(validation.DeviceID, getClientIP(r))

		// Inject device ID into context
		ctx := context.WithValue(r.Context(), authDeviceIDKey, validation.DeviceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getDeviceID extracts device ID from context.
func getDeviceID(ctx context.Context) string {
	if deviceID, ok := ctx.Value(authDeviceIDKey).(string); ok {
		return deviceID
	}
	return ""
}

// handleInfo handles GET /api/lan/info - returns device information.
func (h *LANHandler) handleInfo(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Success bool             `json:"success"`
		Data    *social.DeviceInfo `json:"data"`
	}{
		Success: true,
		Data: &social.DeviceInfo{
			ID:      h.deviceID,
			Name:    h.deviceName,
			Status:  "online",
			Version: h.version,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChat handles POST /api/lan/chat - synchronous chat endpoint.
func (h *LANHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	deviceID := getDeviceID(r.Context())
	if deviceID == "" {
		writeLANError(w, http.StatusUnauthorized, "device ID not found in context")
		return
	}

	// Parse request
	var req social.ChatRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAuthBodyBytes))
	if err := dec.Decode(&req); err != nil {
		writeLANError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate content
	if strings.TrimSpace(req.Content) == "" {
		writeLANError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Default to text type
	if req.Type == "" {
		req.Type = social.MessageTypeText
	}

	// Get or create session
	session := h.sessions.GetOrCreateSession(deviceID)

	// Process message
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	response, err := h.agentLoop.ProcessDirectWithChannel(
		ctx,
		req.Content,
		session.SessionKey,
		"lan",
		deviceID,
	)
	if err != nil {
		log.Printf("LAN chat error: %v", err)
		writeLANError(w, http.StatusInternalServerError, fmt.Sprintf("failed to process message: %v", err))
		return
	}

	// Build response
	chatResponse := struct {
		Success bool               `json:"success"`
		Data    *social.ChatResponse `json:"data"`
	}{
		Success: true,
		Data: &social.ChatResponse{
			ID:        uuid.New().String(),
			Content:   response,
			Timestamp: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse)
}

// handleChatStream handles GET /api/lan/chat/stream - SSE streaming chat endpoint.
func (h *LANHandler) handleChatStream(w http.ResponseWriter, r *http.Request) {
	deviceID := getDeviceID(r.Context())
	if deviceID == "" {
		writeLANError(w, http.StatusUnauthorized, "device ID not found in context")
		return
	}

	// Get content from query parameter
	content := r.URL.Query().Get("content")
	if content == "" {
		writeLANError(w, http.StatusBadRequest, "content query parameter is required")
		return
	}

	// Decode URL-encoded content
	decodedContent, err := url.QueryUnescape(content)
	if err != nil {
		writeLANError(w, http.StatusBadRequest, "invalid content encoding")
		return
	}

	if strings.TrimSpace(decodedContent) == "" {
		writeLANError(w, http.StatusBadRequest, "content cannot be empty")
		return
	}

	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeLANError(w, http.StatusInternalServerError, "SSE not supported")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get or create session
	session := h.sessions.GetOrCreateSession(deviceID)

	// Generate message ID
	messageID := uuid.New().String()

	// Send start event
	h.sendSSEEvent(w, flusher, "start", social.SSEStartData{ID: messageID})

	// Process message
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	response, err := h.agentLoop.ProcessDirectWithChannel(
		ctx,
		decodedContent,
		session.SessionKey,
		"lan",
		deviceID,
	)
	if err != nil {
		log.Printf("LAN chat stream error: %v", err)
		h.sendSSEEvent(w, flusher, "error", social.SSEErrorData{Error: err.Error()})
		return
	}

	// For now, send the complete response as a single chunk
	// In a future implementation, this could be enhanced to stream token by token
	h.sendSSEEvent(w, flusher, "chunk", social.SSEChunkData{Content: response})

	// Send end event
	h.sendSSEEvent(w, flusher, "end", social.SSEEndData{
		Content:   response,
		Timestamp: time.Now(),
	})
}

// sendSSEEvent sends a Server-Sent Event to the client.
func (h *LANHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal SSE data: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", dataJSON)
	flusher.Flush()
}

// writeLANError writes an error response for LAN API.
func writeLANError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// SetAgentLoop updates the agent loop reference.
// This is useful when the agent is initialized after the handler is created.
func (h *LANHandler) SetAgentLoop(loop *agent.AgentLoop) {
	h.agentLoop = loop
}

// GetSessionManager returns the session manager for testing purposes.
func (h *LANHandler) GetSessionManager() *social.SessionManager {
	return h.sessions
}

// NewLANHandlerWithConfig creates a LAN handler with config-based version.
func NewLANHandlerWithConfig(
	deviceStore *devices.DeviceStore,
	agentLoop *agent.AgentLoop,
	deviceID, deviceName string,
) *LANHandler {
	return NewLANHandler(deviceStore, agentLoop, deviceID, deviceName, config.GetVersion())
}
