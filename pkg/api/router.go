package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
)

// Handler is the top-level HTTP handler for the console API.
// It holds references to the agent loop (for chat) and config path (for CRUD).
type Handler struct {
	configPath string
	agent      *agent.AgentLoop
	chatHub    *ChatHub
	tokenStore *tokenStore
	mux        *http.ServeMux
}

// NewHandler creates a new API handler.
// configPath: path to the main MoonHub config.json (provisioning.json is assumed to be alongside it).
// agent: the agent loop for processing chat messages (may be nil during initial setup).
// chatHub: the WebSocket hub for broadcasting agent events (may be nil during initial setup).
func NewHandler(configPath string, agent *agent.AgentLoop, chatHub *ChatHub) *Handler {
	h := &Handler{
		configPath: configPath,
		agent:      agent,
		chatHub:    chatHub,
		tokenStore: newTokenStore(),
		mux:        http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

// SetAgent updates the agent loop reference (called after agent loop is created).
func (h *Handler) SetAgent(agent *agent.AgentLoop) {
	h.agent = agent
}

// SetChatHub updates the chat hub reference.
func (h *Handler) SetChatHub(chatHub *ChatHub) {
	h.chatHub = chatHub
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// configDir returns the directory containing the config file.
// If configPath is itself a directory, it is returned as-is.
func (h *Handler) configDir() string {
	if info, err := os.Stat(h.configPath); err == nil && info.IsDir() {
		return h.configPath
	}
	return filepath.Dir(h.configPath)
}

// RegisterOnMux registers all API routes on an external ServeMux.
// This is called by the channel manager's SetupHTTPServer to share the HTTP server.
func (h *Handler) RegisterOnMux(mux *http.ServeMux) {
	// Auth routes (no middleware)
	mux.HandleFunc("POST /api/auth/bind", h.withoutAuth(h.handleBind))
	mux.HandleFunc("POST /api/auth/refresh", h.withoutAuth(h.handleRefresh))
	mux.HandleFunc("GET /api/auth/status", h.withoutAuth(h.handleAuthStatus))

	// Protected routes (require valid token)
	mux.Handle("GET /api/config", h.authMiddleware(http.HandlerFunc(h.handleGetConfig)))
	mux.Handle("PATCH /api/config", h.authMiddleware(http.HandlerFunc(h.handlePatchConfig)))
	mux.Handle("GET /api/models", h.authMiddleware(http.HandlerFunc(h.handleListModels)))
	mux.Handle("PATCH /api/models/{index}", h.authMiddleware(http.HandlerFunc(h.handleUpdateModel)))
	mux.Handle("GET /api/tools", h.authMiddleware(http.HandlerFunc(h.handleListTools)))
	mux.Handle("GET /api/channels/catalog", h.authMiddleware(http.HandlerFunc(h.handleChannelCatalog)))

	// WebSocket (auth via query param)
	mux.HandleFunc("GET /api/chat/ws", h.handleChatWS)
}

func (h *Handler) registerRoutes() {
	// Same routes registered on internal mux (for standalone testing)
	h.RegisterOnMux(h.mux)
}

func (h *Handler) withoutAuth(next http.HandlerFunc) http.HandlerFunc {
	return next
}
