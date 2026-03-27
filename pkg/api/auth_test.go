package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/provisioning"
	"github.com/stretchr/testify/assert"
)

func setupTestAuth(t *testing.T) (*Handler, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	// Create provisioning config with known auth code
	provPath := filepath.Join(tmpDir, "provisioning.json")
	store, err := provisioning.NewJSONConfigStore(provPath)
	assert.NoError(t, err)
	store.Set(provisioning.KeyAuthCode, "123456")
	store.Set(provisioning.KeyDeviceId, "TEST-001")
	store.Save()

	h := NewHandler(tmpDir, nil, nil)
	return h, func() { os.RemoveAll(tmpDir) }
}

func TestBind_Success(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	body := `{"code":"123456"}`
	req := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp bindResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "TEST-001", resp.DeviceID)
}

func TestBind_WrongCode(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	body := `{"code":"000000"}`
	req := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBind_CodeUsedOnce(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	body := `{"code":"123456"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if i == 0 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		}
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	// Bind first to get a token
	bindReq := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(`{"code":"123456"}`))
	bindReq.Header.Set("Content-Type", "application/json")
	bindW := httptest.NewRecorder()
	h.ServeHTTP(bindW, bindReq)

	var bindResp bindResponse
	json.Unmarshal(bindW.Body.Bytes(), &bindResp)

	// Access protected endpoint
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer "+bindResp.Token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_Success(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	// Bind first to create a binding
	bindReq := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(`{"code":"123456"}`))
	bindReq.Header.Set("Content-Type", "application/json")
	bindW := httptest.NewRecorder()
	h.ServeHTTP(bindW, bindReq)

	var bindResp bindResponse
	json.Unmarshal(bindW.Body.Bytes(), &bindResp)

	// Refresh with the device ID
	refreshReq := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	refreshReq.Header.Set("X-Device-ID", "TEST-001")
	refreshW := httptest.NewRecorder()
	h.ServeHTTP(refreshW, refreshReq)

	assert.Equal(t, http.StatusOK, refreshW.Code)

	var refreshResp map[string]string
	json.Unmarshal(refreshW.Body.Bytes(), &refreshResp)
	assert.NotEmpty(t, refreshResp["token"])
	assert.NotEqual(t, bindResp.Token, refreshResp["token"])
}

func TestRefresh_UnknownDevice(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	req.Header.Set("X-Device-ID", "UNKNOWN")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthStatus_Unbound(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var status map[string]any
	json.Unmarshal(w.Body.Bytes(), &status)
	assert.Equal(t, false, status["bound"])
}

func TestAuthStatus_Bound(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	// Bind first
	bindReq := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(`{"code":"123456"}`))
	bindReq.Header.Set("Content-Type", "application/json")
	bindW := httptest.NewRecorder()
	h.ServeHTTP(bindW, bindReq)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var status map[string]any
	json.Unmarshal(w.Body.Bytes(), &status)
	assert.Equal(t, true, status["bound"])
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/config", nil)
	// No Authorization header at all
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
