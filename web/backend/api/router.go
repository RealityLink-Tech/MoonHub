package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
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

	// Device discovery and pairing
	deviceStore    *devices.DeviceStore
	pairingManager *devices.PairingManager

	// Agent loop for LAN chat
	agentLoop *agent.AgentLoop

	// Device info for LAN API
	deviceID   string
	deviceName string
}

// NewHandler creates an instance of the API handler.
func NewHandler(configPath string) *Handler {
	return &Handler{
		configPath: configPath,
		serverPort: launcherconfig.DefaultPort,
		oauthFlows: make(map[string]*oauthFlow),
		oauthState: make(map[string]string),
	}
}

// SetDeviceStore sets the device store for pairing management.
func (h *Handler) SetDeviceStore(store *devices.DeviceStore) {
	h.deviceStore = store
	h.pairingManager = devices.NewPairingManager(store)
	if _, err := h.pairingManager.GenerateCodeOnce(); err != nil {
		log.Printf("device pairing: failed to ensure pairing code: %v", err)
	}
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

// SetAgentLoop sets the agent loop for LAN chat functionality.
func (h *Handler) SetAgentLoop(loop *agent.AgentLoop) {
	h.agentLoop = loop
}

// SetDeviceInfo sets the device ID and name for LAN API.
func (h *Handler) SetDeviceInfo(id, name string) {
	h.deviceID = id
	h.deviceName = name
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

	// Skills and tools support/actions
	h.registerSkillRoutes(mux)
	h.registerToolRoutes(mux)

	// OS startup / launch-at-login
	h.registerStartupRoutes(mux)

	// Launcher service parameters (port/public)
	h.registerLauncherConfigRoutes(mux)

	// Device discovery (ping, system info)
	if h.deviceStore != nil {
		discoveryHandler := NewDiscoveryHandler(h.deviceStore)
		discoveryHandler.RegisterRoutes(mux)
	}

	// Device pairing and authentication
	if h.pairingManager != nil && h.deviceStore != nil {
		authHandler := NewAuthHandler(h.pairingManager, h.deviceStore)
		authHandler.RegisterRoutes(mux)
	}

	// Provisioning (device management)
	if h.provisioning != nil {
		h.provisioning.RegisterRoutes(mux)
	}

	// LAN chat API (requires agent loop and device store)
	if h.agentLoop != nil && h.deviceStore != nil && h.deviceID != "" {
		lanHandler := NewLANHandlerWithConfig(
			h.deviceStore,
			h.agentLoop,
			h.deviceID,
			h.deviceName,
		)
		lanHandler.RegisterRoutes(mux)
	}
}
