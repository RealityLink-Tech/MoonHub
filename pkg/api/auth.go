package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/provisioning"
)

// ==================== Token Store ====================

type binding struct {
	Token     string    `json:"token"`
	DeviceID  string    `json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
}

type tokenStore struct {
	mu       sync.RWMutex
	tokens   map[string]binding // token -> binding
	bindings map[string]binding // deviceID -> binding
}

func newTokenStore() *tokenStore {
	return &tokenStore{
		tokens:   make(map[string]binding),
		bindings: make(map[string]binding),
	}
}

func (s *tokenStore) Validate(token string) (deviceID string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, exists := s.tokens[token]
	if !exists {
		return "", false
	}
	return b.DeviceID, true
}

func (s *tokenStore) Store(deviceID, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := binding{Token: token, DeviceID: deviceID, CreatedAt: time.Now()}
	s.tokens[token] = b
	s.bindings[deviceID] = b
}

func (s *tokenStore) Refresh(deviceID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bindings[deviceID]; !exists {
		return "", false
	}
	token := generateToken()
	b := binding{Token: token, DeviceID: deviceID, CreatedAt: time.Now()}
	s.tokens[token] = b
	s.bindings[deviceID] = b
	return token, true
}

func (s *tokenStore) HasBindings() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.bindings) > 0
}

// ==================== Auth Handlers ====================

type bindRequest struct {
	Code string `json:"code"`
}

type bindResponse struct {
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
}

func (h *Handler) handleBind(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<10))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Bad request")
		return
	}
	defer r.Body.Close()

	var req bindRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(req.Code) != 6 {
		writeJSONError(w, http.StatusBadRequest, "code must be 6 digits")
		return
	}

	// Load provisioning config
	provPath := filepath.Join(h.configDir(), "provisioning.json")
	provStore, err := provisioning.NewJSONConfigStore(provPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load provisioning config")
		return
	}

	storedCode := ""
	if v := provStore.Get(provisioning.KeyAuthCode); v != nil {
		if s, ok := v.(string); ok {
			storedCode = s
		}
	}
	if storedCode == "" || storedCode != req.Code {
		writeJSONError(w, http.StatusUnauthorized, "invalid auth code")
		return
	}

	deviceID := ""
	if v := provStore.Get(provisioning.KeyDeviceId); v != nil {
		if s, ok := v.(string); ok {
			deviceID = s
		}
	}

	// Invalidate the auth code after successful bind (single-use)
	provStore.Delete(provisioning.KeyAuthCode)
	if err := provStore.Save(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to persist auth code invalidation")
		return
	}

	// Issue token
	token := generateToken()
	h.tokenStore.Store(deviceID, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bindResponse{
		Token:    token,
		DeviceID: deviceID,
	})
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	deviceID := r.Header.Get("X-Device-Id")
	if deviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "X-Device-Id header required")
		return
	}

	token, ok := h.tokenStore.Refresh(deviceID)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "no binding found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"bound": h.tokenStore.HasBindings(),
	})
}

// ==================== Auth Middleware ====================

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			writeJSONError(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		token := auth[7:]

		deviceID, ok := h.tokenStore.Validate(token)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Store device ID in request context for downstream handlers
		r = r.WithContext(contextWithDeviceID(r.Context(), deviceID))
		next.ServeHTTP(w, r)
	})
}

// ==================== Helpers ====================

type contextKey string

const deviceIDKey contextKey = "device_id"

func contextWithDeviceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, deviceIDKey, id)
}

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// writeJSONError writes a JSON-encoded error response with the given HTTP status.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
