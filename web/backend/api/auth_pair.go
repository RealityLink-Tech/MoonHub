package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
	"github.com/google/uuid"
)

const maxAuthBodyBytes = 64 * 1024

// AuthHandler handles device pairing and authentication API requests.
type AuthHandler struct {
	pairingManager *devices.PairingManager
	deviceStore    *devices.DeviceStore
}

// NewAuthHandler creates a new auth API handler.
func NewAuthHandler(pairingManager *devices.PairingManager, deviceStore *devices.DeviceStore) *AuthHandler {
	return &AuthHandler{
		pairingManager: pairingManager,
		deviceStore:    deviceStore,
	}
}

// RegisterRoutes registers auth routes with the mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/pair", h.handleAuthPair)
	mux.HandleFunc("POST /api/auth/verify", h.handleAuthVerify)
}

// AuthStatusResponse represents the auth status response.
type AuthStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Code   string `json:"code"`
		Paired bool   `json:"paired"`
	} `json:"data"`
}

// PairRequest represents a pairing request.
type PairRequest struct {
	Code       string `json:"code"`
	DeviceID   string `json:"deviceId,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
}

// UnmarshalJSON accepts camelCase fields and snake_case aliases (device_id, device_name).
func (p *PairRequest) UnmarshalJSON(data []byte) error {
	type wire struct {
		Code          string `json:"code"`
		DeviceID      string `json:"deviceId"`
		DeviceName    string `json:"deviceName"`
		DeviceIDSnake string `json:"device_id"`
		DeviceNameSnk string `json:"device_name"`
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	p.Code = w.Code
	p.DeviceID = w.DeviceID
	if p.DeviceID == "" {
		p.DeviceID = w.DeviceIDSnake
	}
	p.DeviceName = w.DeviceName
	if p.DeviceName == "" {
		p.DeviceName = w.DeviceNameSnk
	}
	return nil
}

// PairResponse represents a pairing response.
type PairResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Token          string                `json:"token,omitempty"`
		TokenExpiresAt time.Time             `json:"tokenExpiresAt,omitempty"`
		Device         *devices.PairedDevice `json:"device,omitempty"`
	} `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// VerifyResponse represents a token verification response.
type VerifyResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Valid     bool       `json:"valid"`
		DeviceID  string     `json:"deviceId,omitempty"`
		ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	} `json:"data"`
}

// handleAuthStatus handles GET /api/auth/status - returns current pairing code status.
func (h *AuthHandler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	code := h.pairingManager.GetCurrentCode()
	paired := h.pairingManager.IsPaired()

	// If already paired, don't show the code
	if paired {
		code = ""
	}

	response := AuthStatusResponse{
		Success: true,
	}
	response.Data.Code = code
	response.Data.Paired = paired

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAuthPair handles POST /api/auth/pair - pairs a device with authorization code.
func (h *AuthHandler) handleAuthPair(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	// Parse request
	var req PairRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAuthBodyBytes))
	if err := dec.Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate code
	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))
	if req.Code == "" {
		writeAuthError(w, http.StatusBadRequest, "pairing code is required")
		return
	}

	// Generate device ID if not provided
	if req.DeviceID == "" {
		req.DeviceID = uuid.New().String()
	}

	// Generate device name if not provided
	deviceName := req.DeviceName
	if deviceName == "" {
		deviceName = "PWA Client"
	}

	// Get client IP
	ipAddr := getClientIP(r)

	// Create device info
	deviceInfo := &devices.PairedDevice{
		ID:        req.DeviceID,
		Name:      deviceName,
		IPAddress: ipAddr,
		UserAgent: r.UserAgent(),
	}

	// Validate pairing code
	result, err := h.pairingManager.ValidateCode(req.Code, deviceInfo)
	if err != nil {
		log.Printf("pairing error: %v", err)
		writeAuthError(w, http.StatusInternalServerError, "pairing failed")
		return
	}

	if !result.Success {
		writeAuthError(w, http.StatusBadRequest, result.Error)
		return
	}

	// Return success with token
	response := PairResponse{
		Success: true,
	}
	response.Data.Token = result.Token
	response.Data.TokenExpiresAt = result.TokenExpiresAt
	response.Data.Device = result.Device

	log.Printf("device paired: %s (%s)", deviceInfo.ID, deviceInfo.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAuthVerify handles POST /api/auth/verify - verifies a Bearer token.
func (h *AuthHandler) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	token := ""
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = strings.TrimSpace(parts[1])
		}
	}

	if token == "" {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxAuthBodyBytes))
		if err == nil && len(strings.TrimSpace(string(body))) > 0 {
			var payload struct {
				Token string `json:"token"`
			}
			if json.Unmarshal(body, &payload) == nil {
				token = strings.TrimSpace(payload.Token)
			}
		}
	}

	if token == "" {
		writeAuthError(w, http.StatusUnauthorized, "token required (Authorization: Bearer or JSON {\"token\"})")
		return
	}

	// Validate token
	validation := h.deviceStore.ValidateToken(token)

	response := VerifyResponse{
		Success: true,
	}
	response.Data.Valid = validation.Valid

	if validation.Valid {
		response.Data.DeviceID = validation.DeviceID
		response.Data.ExpiresAt = validation.ExpiresAt

		// Update last seen
		_ = h.deviceStore.UpdateLastSeen(validation.DeviceID, getClientIP(r))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeAuthError writes an error response.
func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// getClientIP extracts the client IP string for persistence (paired device last-seen).
func getClientIP(r *http.Request) string {
	ip := ClientIPFromRequest(r)
	if ip == nil {
		return ""
	}
	return ip.String()
}
