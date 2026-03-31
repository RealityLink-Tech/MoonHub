package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
	"github.com/RealityLink-Tech/MoonHub/pkg/mdns"
	pkgapi "github.com/RealityLink-Tech/MoonHub/pkg/api"
	"github.com/RealityLink-Tech/MoonHub/web/backend/launcherconfig"
)

// Handler serves HTTP API requests.
type Handler struct {
	configPath           string
	serverPort           int
	serverPublic         bool
	serverPublicExplicit bool
	serverCIDRs          []string
	oauthMu              sync.Mutex
	oauthFlows           map[string]*oauthFlow
	oauthState           map[string]string
	provisioning         *ProvisioningHandler
	discovery            *DiscoveryHandler
	discover             *DiscoverHandler
	devices              *DevicesHandler
	auth                 *AuthHandler
	dynamicTools         *DynamicToolsHandler
	agentLoop            *agent.AgentLoop
	chatHub              *pkgapi.ChatHub
	chat                 *ChatHandler
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string, deviceStore *devices.DeviceStore, pairingManager *devices.PairingManager) *Handler {
	mdnsClient := mdns.NewClient()
	h := &Handler{
		configPath:   configPath,
		serverPort:   launcherconfig.DefaultPort,
		oauthFlows:   make(map[string]*oauthFlow),
		oauthState:   make(map[string]string),
		discovery:    NewDiscoveryHandler(deviceStore),
		discover:     NewDiscoverHandler(mdnsClient),
		devices:      NewDevicesHandler(deviceStore),
		auth:         NewAuthHandler(pairingManager, deviceStore),
		dynamicTools: initDynamicTools(configPath),
	}
	h.chatHub = pkgapi.NewChatHub(nil)
	h.chat = NewChatHandler(deviceStore, h.chatHub)
	return h
}

// SetServerOptions stores current backend listen options for fallback behavior.
func (h *Handler) SetServerOptions(port int, public bool, publicExplicit bool, allowedCIDRs []string) {
	h.serverPort = port
	h.serverPublic = public
	h.serverPublicExplicit = publicExplicit
	h.serverCIDRs = append([]string(nil), allowedCIDRs...)
}

// SetProvisioningHandler sets the provisioning handler for device management.
func (h *Handler) SetProvisioningHandler(handler *ProvisioningHandler) {
	h.provisioning = handler
}

// SetAgentLoop sets the in-process agent loop for chat functionality.
func (h *Handler) SetAgentLoop(loop *agent.AgentLoop) {
	h.agentLoop = loop
	h.chatHub.SetAgent(loop)
	h.chat.SetAgentLoop(loop)
}

// AgentLoop returns the in-process agent loop (nil if not running single-process).
func (h *Handler) AgentLoop() *agent.AgentLoop {
	return h.agentLoop
}

// RegisterRoutes binds all API endpoint handlers to the ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Config CRUD
	h.registerConfigRoutes(mux)

	// Gateway process lifecycle
	h.registerGatewayRoutes(mux)

	// Session history
	h.registerSessionRoutes(mux)

	// OAuth login and credential management
	h.registerOAuthRoutes(mux)

	// Model list management
	h.registerModelRoutes(mux)

	// Channel catalog (for frontend navigation/config pages)
	h.registerChannelRoutes(mux)

	// Channel CRUD (create, read, update, delete channel configurations)
	h.registerChannelCRUDRoutes(mux)

	// Skills and tools support/actions
	h.registerSkillRoutes(mux)
	h.registerToolRoutes(mux)

	// OS startup / launch-at-login
	h.registerStartupRoutes(mux)

	// Launcher service parameters (port/public)
	h.registerLauncherConfigRoutes(mux)

	// Chat routes (REST, SSE, WebSocket)
	h.chat.RegisterRoutes(mux)

	// Provisioning (device management)
	if h.provisioning != nil {
		h.provisioning.RegisterRoutes(mux)
	}

	// Device discovery and authentication
	h.discovery.RegisterRoutes(mux)
	h.discover.RegisterRoutes(mux)
	h.devices.RegisterRoutes(mux)
	h.auth.RegisterRoutes(mux)

	// Dynamic tools (AI-generated UI components)
	if h.dynamicTools != nil {
		h.dynamicTools.RegisterRoutes(mux)
	}
}

// initDynamicTools creates the DynamicToolsHandler. If initialisation fails
// (e.g. SQLite unavailable) the handler is nil and routes are silently skipped.
func initDynamicTools(configPath string) *DynamicToolsHandler {
	dth, err := NewDynamicToolsHandler(configPath)
	if err != nil {
		log.Printf("dynamic-tools: init skipped: %v", err)
		return nil
	}
	return dth
}
