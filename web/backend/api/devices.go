package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// PublicPairedDevice is the public representation of a paired device,
// with sensitive fields (Token, TokenExpiresAt) stripped.
type PublicPairedDevice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
	PairedAt  time.Time `json:"pairedAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// toPublic maps a PairedDevice to its public-safe representation.
func toPublicPairedDevice(p *devices.PairedDevice) PublicPairedDevice {
	return PublicPairedDevice{
		ID:        p.ID,
		Name:      p.Name,
		IPAddress: p.IPAddress,
		UserAgent: p.UserAgent,
		PairedAt:  p.PairedAt,
		LastSeenAt: p.LastSeenAt,
	}
}

// deviceStoreInterface defines the interface for device store operations.
// This allows for mocking in tests.
type deviceStoreInterface interface {
	ListDevices() []*devices.PairedDevice
}

// DevicesHandler handles paired devices API requests.
type DevicesHandler struct {
	deviceStore deviceStoreInterface
}

// NewDevicesHandler creates a new devices API handler.
func NewDevicesHandler(store *devices.DeviceStore) *DevicesHandler {
	return &DevicesHandler{
		deviceStore: store,
	}
}

// RegisterRoutes registers devices routes with the mux.
func (h *DevicesHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/devices", h.handleListDevices)
}

// handleListDevices handles GET /api/devices - returns all paired devices.
func (h *DevicesHandler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	// Enforce LAN-only restriction
	if !requireLANClient(w, r) {
		return
	}

	// Get all paired devices from store and strip sensitive fields
	storedDevices := h.deviceStore.ListDevices()
	publicDevices := make([]PublicPairedDevice, len(storedDevices))
	for i, d := range storedDevices {
		publicDevices[i] = toPublicPairedDevice(d)
	}

	// Return successful response with device list
	response := map[string]interface{}{
		"success": true,
		"data":    publicDevices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
