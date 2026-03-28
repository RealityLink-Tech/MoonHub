package api

import (
	"encoding/json"
	"net/http"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

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

	// Get all paired devices from store
	devices := h.deviceStore.ListDevices()

	// Return successful response with device list
	response := map[string]interface{}{
		"success": true,
		"data":    devices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
