package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/provisioning"
)

const maxProvisioningBodyBytes = 64 * 1024

// ProvisioningHandler handles device provisioning API requests.
type ProvisioningHandler struct {
	manager *provisioning.DeviceManager
}

// NewProvisioningHandler creates a new provisioning API handler.
func NewProvisioningHandler(manager *provisioning.DeviceManager) *ProvisioningHandler {
	return &ProvisioningHandler{
		manager: manager,
	}
}

// RegisterRoutes registers provisioning routes with the mux.
func (h *ProvisioningHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/provisioning/status", h.handleGetStatus)
	mux.HandleFunc("GET /api/provisioning/events", h.handleEvents)
	mux.HandleFunc("GET /api/provisioning/networks", h.handleListNetworks)
	mux.HandleFunc("GET /api/provisioning/networks/saved", h.handleListSavedNetworks)
	mux.HandleFunc("POST /api/provisioning/network/connect", h.handleConnectWiFi)
	mux.HandleFunc("POST /api/provisioning/network/forget", h.handleForgetNetwork)
	mux.HandleFunc("GET /api/provisioning/network/diagnostics", h.handleDiagnostics)
	mux.HandleFunc("POST /api/provisioning/network/test", h.handleTestInternet)
	mux.HandleFunc("POST /api/provisioning/hotspot/enable", h.handleEnableHotspot)
	mux.HandleFunc("POST /api/provisioning/hotspot/disable", h.handleDisableHotspot)
	mux.HandleFunc("POST /api/provisioning/restart", h.handleRestart)
	mux.HandleFunc("POST /api/provisioning/rollback", h.handleRollback)
	mux.HandleFunc("GET /api/provisioning/auth-code", h.handleGetAuthCode)
	mux.HandleFunc("POST /api/provisioning/auth-code/regenerate", h.handleRegenerateAuthCode)
	mux.HandleFunc("POST /api/provisioning/recovery/trigger", h.handleTriggerRecovery)
	mux.HandleFunc("POST /api/provisioning/factory-reset", h.handleFactoryReset)
}

func (h *ProvisioningHandler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.manager.GetStatus()
	if err != nil {
		log.Printf("provisioning: get status: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": status,
	})
}

func (h *ProvisioningHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	events := h.manager.GetEvents()
	ch := events.Subscribe()
	defer events.Unsubscribe(ch)

	fmt.Fprintf(w, "event: connected\ndata: {\"ok\":true,\"ts\":%d}\n\n", time.Now().UnixMilli())
	flusher.Flush()

	for _, event := range events.GetRecentEvents() {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("provisioning: marshal event: %v", err)
			continue
		}
		fmt.Fprintf(w, "event: device\ndata: %s\n\n", string(data))
		flusher.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: device\ndata: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}

func (h *ProvisioningHandler) handleListNetworks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	networks, err := h.manager.ListWifiNetworks(ctx)
	if err != nil {
		log.Printf("provisioning: list networks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"networks": networks,
	})
}

func (h *ProvisioningHandler) handleListSavedNetworks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	networks, err := h.manager.ListSavedNetworks(ctx)
	if err != nil {
		log.Printf("provisioning: list saved networks: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"networks": networks,
	})
}

func (h *ProvisioningHandler) handleConnectWiFi(w http.ResponseWriter, r *http.Request) {
	var req provisioning.WiFiConnectRequest
	if err := decodeProvisioningJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := h.manager.ConnectToWiFi(ctx, req.SSID, req.Password, req.Hidden); err != nil {
		log.Printf("provisioning: connect wifi: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (h *ProvisioningHandler) handleForgetNetwork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeProvisioningJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.manager.ForgetNetwork(ctx, req.Name); err != nil {
		log.Printf("provisioning: forget network: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (h *ProvisioningHandler) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	diagnostics := h.manager.DiagnoseNetwork(ctx)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"diagnostics": diagnostics,
	})
}

func (h *ProvisioningHandler) handleTestInternet(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	connected := h.manager.TestInternet(ctx)
	diagnostics := h.manager.DiagnoseNetwork(ctx)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connected":   connected,
		"diagnostics": diagnostics,
	})
}

func (h *ProvisioningHandler) handleEnableHotspot(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.manager.EnableHotspot(ctx); err != nil {
		log.Printf("provisioning: enable hotspot: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (h *ProvisioningHandler) handleDisableHotspot(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.manager.DisableHotspot(ctx); err != nil {
		log.Printf("provisioning: disable hotspot: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (h *ProvisioningHandler) handleRestart(w http.ResponseWriter, r *http.Request) {
	var req provisioning.RestartRequest
	if err := decodeProvisioningJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.manager.RequestRestart(ctx, req.Reason, req.Scope); err != nil {
		log.Printf("provisioning: restart: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (h *ProvisioningHandler) handleRollback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeProvisioningJSON(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.manager.RollbackToProvisioning(ctx, req.Reason); err != nil {
		log.Printf("provisioning: rollback: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "network operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": message,
	})
}

func decodeProvisioningJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProvisioningBodyBytes))
	return dec.Decode(v)
}

func (h *ProvisioningHandler) handleGetAuthCode(w http.ResponseWriter, r *http.Request) {
	code, deviceID, err := h.manager.GetOrCreateAuthCode()
	if err != nil {
		log.Printf("provisioning: auth code: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":     code,
		"deviceId": deviceID,
	})
}

func (h *ProvisioningHandler) handleRegenerateAuthCode(w http.ResponseWriter, r *http.Request) {
	code, deviceID, err := h.manager.RegenerateAuthCode()
	if err != nil {
		log.Printf("provisioning: regenerate auth code: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":     code,
		"deviceId": deviceID,
	})
}

func (h *ProvisioningHandler) handleTriggerRecovery(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := h.manager.TriggerRecovery(ctx); err != nil {
		log.Printf("provisioning: trigger recovery: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (h *ProvisioningHandler) handleFactoryReset(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := h.manager.FactoryReset(ctx); err != nil {
		log.Printf("provisioning: factory reset: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}
