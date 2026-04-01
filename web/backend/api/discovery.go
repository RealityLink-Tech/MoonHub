package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// DiscoveryHandler handles device discovery API requests.
type DiscoveryHandler struct {
	deviceStore *devices.DeviceStore
	startTime   time.Time
}

// NewDiscoveryHandler creates a new discovery API handler.
func NewDiscoveryHandler(deviceStore *devices.DeviceStore) *DiscoveryHandler {
	return &DiscoveryHandler{
		deviceStore: deviceStore,
		startTime:   time.Now(),
	}
}

// RegisterRoutes registers discovery routes with the mux.
func (h *DiscoveryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ping", h.handlePing)
	mux.HandleFunc("GET /api/system/info", h.handleSystemInfo)
}

// handlePing handles GET /api/ping - simple device presence check.
func (h *DiscoveryHandler) handlePing(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"version": config.GetVersion(),
			"name":    "MoonHub Device",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	writeCORSHeaders(w, r)
	json.NewEncoder(w).Encode(response)
}

// handleSystemInfo handles GET /api/system/info - detailed device status.
func (h *DiscoveryHandler) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	uptime := int64(time.Since(h.startTime).Seconds())

	status := devices.DeviceStatus{
		Version:     config.GetVersion(),
		Platform:    runtime.GOOS,
		Arch:        runtime.GOARCH,
		Uptime:      uptime,
		Paired:      h.deviceStore.HasDevices(),
		PairedCount: h.deviceStore.Count(),
	}

	response := map[string]interface{}{
		"success": true,
		"data":    status,
	}

	w.Header().Set("Content-Type", "application/json")
	writeCORSHeaders(w, r)
	json.NewEncoder(w).Encode(response)
}
