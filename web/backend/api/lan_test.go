package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
	"github.com/RealityLink-Tech/MoonHub/pkg/social"
)

// addTestDevice creates and stores a test device with a valid token.
func addTestDevice(t *testing.T, store *devices.DeviceStore, deviceID, name string) string {
	t.Helper()
	token, _, _ := devices.GenerateToken()
	_ = store.AddDevice(&devices.PairedDevice{
		ID:             deviceID,
		Name:           name,
		Token:          token,
		TokenExpiresAt: time.Now().Add(24 * time.Hour),
		PairedAt:       time.Now(),
		IPAddress:      "127.0.0.1",
	})
	return token
}

// newTestLANHandler creates a LANHandler with mocked dependencies for testing.
func newTestLANHandler(t *testing.T) (*LANHandler, *devices.DeviceStore) {
	t.Helper()
	store, _ := devices.NewDeviceStore(t.TempDir())
	h := &LANHandler{
		deviceStore: store,
		deviceID:    "test-device-id",
		deviceName:  "Test Device",
		version:     "1.0.0-test",
		sessions:    social.NewSessionManager(),
	}
	return h, store
}

func TestHandleInfo_NoAuth(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleInfo_WithValidToken(t *testing.T) {
	h, store := newTestLANHandler(t)
	token := addTestDevice(t, store, "client-1", "Test Client")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Data.ID != "test-device-id" {
		t.Errorf("expected device ID test-device-id, got %s", resp.Data.ID)
	}
	if resp.Data.Name != "Test Device" {
		t.Errorf("expected name Test Device, got %s", resp.Data.Name)
	}
	if resp.Data.Status != "online" {
		t.Errorf("expected status online, got %s", resp.Data.Status)
	}
	if resp.Data.Version != "1.0.0-test" {
		t.Errorf("expected version 1.0.0-test, got %s", resp.Data.Version)
	}
}

func TestHandleChat_NoAuth(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"type":"text","content":"hello"}`
	req := httptest.NewRequest("POST", "/api/lan/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleChat_NoContent(t *testing.T) {
	h, store := newTestLANHandler(t)
	token := addTestDevice(t, store, "client-1", "Test Client")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"type":"text","content":"  "}`
	req := httptest.NewRequest("POST", "/api/lan/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleChat_InvalidBody(t *testing.T) {
	h, store := newTestLANHandler(t)
	token := addTestDevice(t, store, "client-1", "Test Client")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/lan/chat", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleChatStream_NoContent(t *testing.T) {
	h, store := newTestLANHandler(t)
	token := addTestDevice(t, store, "client-1", "Test Client")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/chat/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleChatStream_NoAuth(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/chat/stream?content=hello", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleInfo_InvalidTokenFormat(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.Header.Set("Authorization", "Bearer invalid-short")
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_NoAuthorizationHeader(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.Header.Set("Authorization", "Basic abc123")
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestNewLANHandler(t *testing.T) {
	store, _ := devices.NewDeviceStore(t.TempDir())
	h := NewLANHandler(store, nil, "dev-1", "My Device", "2.0.0")

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.deviceID != "dev-1" {
		t.Errorf("expected device ID dev-1, got %s", h.deviceID)
	}
	if h.deviceName != "My Device" {
		t.Errorf("expected name My Device, got %s", h.deviceName)
	}
	if h.version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", h.version)
	}
	if h.agentLoop != nil {
		t.Error("expected nil agentLoop")
	}
	if h.GetSessionManager() == nil {
		t.Error("expected non-nil session manager")
	}
	if h.GetSessionManager().Count() != 0 {
		t.Error("expected 0 sessions")
	}
}

func TestSetAgentLoop(t *testing.T) {
	store, _ := devices.NewDeviceStore(t.TempDir())
	h := NewLANHandler(store, nil, "dev-1", "My Device", "1.0.0")

	if h.agentLoop != nil {
		t.Error("expected nil agentLoop initially")
	}

	h.SetAgentLoop(nil) // just test it doesn't panic
}

func TestWriteLANError(t *testing.T) {
	w := httptest.NewRecorder()
	writeLANError(w, http.StatusForbidden, "access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error != "access denied" {
		t.Errorf("expected error 'access denied', got %s", resp.Error)
	}
}

func TestSendSSEEvent(t *testing.T) {
	w := httptest.NewRecorder()
	h := &LANHandler{}
	h.sendSSEEvent(w, w, "chunk", social.SSEChunkData{Content: "hello"})

	body := w.Body.String()
	if !strings.Contains(body, "event: chunk") {
		t.Errorf("expected 'event: chunk' in output, got: %s", body)
	}
	if !strings.Contains(body, `"content":"hello"`) {
		t.Errorf("expected content in output, got: %s", body)
	}
}

func TestHandleChatStream_EmptyContent(t *testing.T) {
	h, store := newTestLANHandler(t)
	token := addTestDevice(t, store, "client-1", "Test Client")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/chat/stream?content=", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty content, got %d", w.Code)
	}
}

func TestHandleInfo_ResponseContentType(t *testing.T) {
	h, store := newTestLANHandler(t)
	token := addTestDevice(t, store, "client-1", "Test Client")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content type, got %s", ct)
	}
}

func TestRegisterRoutes(t *testing.T) {
	h, _ := newTestLANHandler(t)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Verify routes are registered by checking 404 for unregistered paths
	req := httptest.NewRequest("GET", "/api/lan/nonexistent", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unregistered route, got %d", w.Code)
	}
}

func TestHandleChat_ExpiredToken(t *testing.T) {
	store, _ := devices.NewDeviceStore(t.TempDir())
	token, _, _ := devices.GenerateToken()
	// Add device with expired token
	_ = store.AddDevice(&devices.PairedDevice{
		ID:             "client-1",
		Name:           "Test Client",
		Token:          token,
		TokenExpiresAt: time.Now().Add(-1 * time.Hour), // expired
		PairedAt:       time.Now(),
		IPAddress:      "127.0.0.1",
	})

	h := &LANHandler{
		deviceStore: store,
		deviceID:    "test-device-id",
		deviceName:  "Test Device",
		version:     "1.0.0-test",
		sessions:    social.NewSessionManager(),
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/lan/info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}
