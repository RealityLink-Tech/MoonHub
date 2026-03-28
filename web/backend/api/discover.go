package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/mdns"
)

// mdnsClientInterface defines the interface for mDNS client operations.
// This allows for mocking in tests.
type mdnsClientInterface interface {
	Discover(ctx context.Context) ([]mdns.DeviceInfo, error)
}

// DiscoverHandler handles mDNS device discovery API requests.
type DiscoverHandler struct {
	mdnsClient mdnsClientInterface
}

// NewDiscoverHandler creates a new discover API handler.
func NewDiscoverHandler(client *mdns.Client) *DiscoverHandler {
	return &DiscoverHandler{
		mdnsClient: client,
	}
}

// RegisterRoutes registers discovery routes with the mux.
func (h *DiscoverHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/discover", h.handleDiscover)
}

// handleDiscover handles GET /api/discover - scans LAN for MoonHub devices.
func (h *DiscoverHandler) handleDiscover(w http.ResponseWriter, r *http.Request) {
	// Enforce LAN-only restriction
	if !requireLANClient(w, r) {
		return
	}

	// Create context with 5 second timeout for mDNS discovery
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Perform mDNS discovery
	devices, err := h.mdnsClient.Discover(ctx)
	if err != nil {
		// Context timeout/cancellation is not an error - just return empty results
		if ctx.Err() != nil {
			devices = []mdns.DeviceInfo{}
		} else {
			writeAuthError(w, http.StatusInternalServerError, "discovery failed")
			return
		}
	}

	// Return successful response with device list
	response := map[string]interface{}{
		"success": true,
		"data":    devices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
