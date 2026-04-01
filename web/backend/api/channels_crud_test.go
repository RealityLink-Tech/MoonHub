package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// setupChannelTestEnv creates a test environment with config and handler.
// Returns (configPath, handler, bearerToken, cleanup).
func setupChannelTestEnv(t *testing.T) (string, *Handler, string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create default config
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	deviceStore, err := devices.NewDeviceStore(tmpDir)
	if err != nil {
		t.Fatalf("NewDeviceStore() error = %v", err)
	}
	pairingManager := devices.NewPairingManager(deviceStore)
	token := setupAuthenticatedDevice(t, deviceStore)

	h := NewHandler(configPath, deviceStore, pairingManager)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	cleanup := func() {
		// tmpDir will be cleaned up automatically by t.TempDir()
	}

	return configPath, h, token, cleanup
}

// newLocalRequest creates a request with localhost IP to pass LAN restriction.
func newLocalRequest(method, url string, body *bytes.Reader) *http.Request {
	var bodyReader io.Reader = body
	if body == nil {
		bodyReader = nil
	}
	req := httptest.NewRequest(method, url, bodyReader)
	req.RemoteAddr = "127.0.0.1:12345" // Localhost IP to pass LAN restriction
	return req
}

// TestHandleListChannels_Success tests listing channels when some are configured.
func TestHandleListChannels_Success(t *testing.T) {
	configPath, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// Configure a telegram channel
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "test-token"
	cfg.Channels.Telegram.BaseURL = "https://api.telegram.org/bot"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Test listing channels
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	req.RemoteAddr = "127.0.0.1:12345" // Localhost IP to pass LAN restriction
	h.RegisterRoutes(http.NewServeMux())
	h.handleListChannels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    []channelInstance `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	if len(resp.Data) == 0 {
		t.Fatal("len(data) = 0, want at least 1 (telegram)")
	}

	// Check telegram channel is present
	var telegram *channelInstance
	for i := range resp.Data {
		if resp.Data[i].ID == "telegram" {
			telegram = &resp.Data[i]
			break
		}
	}

	if telegram == nil {
		t.Fatal("telegram channel not found in response")
	}

	if !telegram.Enabled {
		t.Error("telegram channel enabled = false, want true")
	}

	if telegram.Config["base_url"] != "https://api.telegram.org/bot" {
		t.Errorf("telegram config base_url = %v, want https://api.telegram.org/bot", telegram.Config["base_url"])
	}
}

// TestHandleListChannels_Empty tests listing channels when none are configured.
func TestHandleListChannels_Empty(t *testing.T) {
	_, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodGet, "/api/channels", nil)
	h.handleListChannels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    []channelInstance `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	if len(resp.Data) != 0 {
		t.Fatalf("len(data) = %d, want 0 (no channels configured)", len(resp.Data))
	}
}

// TestHandleListChannels_LANRestriction tests that listing channels requires LAN access.
func TestHandleListChannels_LANRestriction(t *testing.T) {
	_, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	// Simulate non-LAN IP by setting RemoteAddr to a public IP
	req.RemoteAddr = "8.8.8.8:12345"
	h.handleListChannels(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (forbidden from non-LAN)", rec.Code, http.StatusForbidden)
	}
}

// TestHandleCreateChannel_Success tests creating a new channel configuration.
func TestHandleCreateChannel_Success(t *testing.T) {
	configPath, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	createReq := channelCreateRequest{
		ID: "telegram",
		Config: map[string]any{
			"token":    "new-test-token",
			"base_url": "https://api.telegram.org/bot",
		},
	}

	body, _ := json.Marshal(createReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.handleCreateChannel(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	if resp.Data.ID != "telegram" {
		t.Errorf("id = %s, want telegram", resp.Data.ID)
	}

	// Verify channel was actually created in config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Channels.Telegram.Enabled {
		t.Error("telegram channel not enabled in config after creation")
	}

	if cfg.Channels.Telegram.Token != "new-test-token" {
		t.Errorf("telegram token = %s, want new-test-token", cfg.Channels.Telegram.Token)
	}
}

// TestHandleCreateChannel_TypeFieldAlias tests PWA-style body: type + name + config (no id).
func TestHandleCreateChannel_TypeFieldAlias(t *testing.T) {
	configPath, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	body := []byte(`{"type":"discord","name":"Team Discord","config":{"token":"discord-token-xyz"}}`)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.handleCreateChannel(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg.Channels.Discord.Enabled {
		t.Fatal("discord channel not enabled after create with type field")
	}
	if cfg.Channels.Discord.Token != "discord-token-xyz" {
		t.Errorf("discord token = %q, want discord-token-xyz", cfg.Channels.Discord.Token)
	}
}

// TestHandleCreateChannel_MissingIDAndType rejects empty id and type.
func TestHandleCreateChannel_MissingIDAndType(t *testing.T) {
	_, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	body := []byte(`{"name":"x","config":{"token":"t"}}`)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.handleCreateChannel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

// TestHandleCreateChannel_InvalidType tests creating a channel with invalid type.
func TestHandleCreateChannel_InvalidType(t *testing.T) {
	_, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	createReq := channelCreateRequest{
		ID: "invalid_channel_type",
		Config: map[string]any{
			"token": "test-token",
		},
	}

	body, _ := json.Marshal(createReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.handleCreateChannel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Success {
		t.Error("success = true, want false for invalid channel type")
	}

	if resp.Error == "" {
		t.Error("error message is empty, want error message")
	}
}

// TestHandleCreateChannel_AlreadyExists tests creating a channel that already exists.
func TestHandleCreateChannel_AlreadyExists(t *testing.T) {
	configPath, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// First, create a telegram channel
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "existing-token"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Try to create it again
	createReq := channelCreateRequest{
		ID: "telegram",
		Config: map[string]any{
			"token": "new-token",
		},
	}

	body, _ := json.Marshal(createReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.handleCreateChannel(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (conflict), body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Success {
		t.Error("success = true, want false when channel already exists")
	}
}

// TestHandleUpdateChannel_Success tests updating an existing channel.
func TestHandleUpdateChannel_Success(t *testing.T) {
	configPath, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// First, create a telegram channel
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "old-token"
	cfg.Channels.Telegram.BaseURL = "https://old-base-url.com"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Update the channel
	updateReq := channelUpdateRequest{
		Config: map[string]any{
			"token":    "new-token",
			"base_url": "https://new-base-url.com",
		},
	}

	body, _ := json.Marshal(updateReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPatch, "/api/channels/telegram", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	withBearerToken(req, token)

	// Register routes and use mux to handle path parameters
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	// Verify config was updated
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Channels.Telegram.Token != "new-token" {
		t.Errorf("telegram token = %s, want new-token", cfg.Channels.Telegram.Token)
	}

	if cfg.Channels.Telegram.BaseURL != "https://new-base-url.com" {
		t.Errorf("telegram base_url = %s, want https://new-base-url.com", cfg.Channels.Telegram.BaseURL)
	}
}

// TestHandleUpdateChannel_NotFound tests updating a channel that doesn't exist.
func TestHandleUpdateChannel_NotFound(t *testing.T) {
	_, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	updateReq := channelUpdateRequest{
		Config: map[string]any{
			"token": "test-token",
		},
	}

	body, _ := json.Marshal(updateReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPatch, "/api/channels/telegram", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	withBearerToken(req, token)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (not found), body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Success {
		t.Error("success = true, want false when channel not found")
	}
}

// TestHandleDeleteChannel_Success tests deleting a channel.
func TestHandleDeleteChannel_Success(t *testing.T) {
	configPath, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// First, create a telegram channel
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "test-token"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Delete the channel
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodDelete, "/api/channels/telegram", nil)
	withBearerToken(req, token)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	// Verify channel was disabled in config
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Channels.Telegram.Enabled {
		t.Error("telegram channel still enabled in config after deletion")
	}
}

// TestHandleDeleteChannel_NotFound tests deleting a channel that doesn't exist.
func TestHandleDeleteChannel_NotFound(t *testing.T) {
	_, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodDelete, "/api/channels/telegram", nil)
	withBearerToken(req, token)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (not found), body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Success {
		t.Error("success = true, want false when channel not found")
	}
}

// TestHandleGetChannelStatus_Success tests getting status of a channel.
func TestHandleGetChannelStatus_Success(t *testing.T) {
	configPath, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// Create a telegram channel
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "test-token"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Get channel status
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodGet, "/api/channels/telegram/status", nil)
	withBearerToken(req, token)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID     string         `json:"id"`
			Status map[string]any `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !resp.Success {
		t.Fatalf("success = false, want true")
	}

	if resp.Data.ID != "telegram" {
		t.Errorf("id = %s, want telegram", resp.Data.ID)
	}

	enabled, ok := resp.Data.Status["enabled"].(bool)
	if !ok || !enabled {
		t.Error("status.enabled = false, want true")
	}

	// running should be false since web backend doesn't have runtime access
	running, ok := resp.Data.Status["running"].(bool)
	if !ok || running {
		t.Error("status.running = true, want false (web backend has no runtime access)")
	}
}

// TestHandleGetChannelStatus_NotFound tests getting status of a non-existent channel.
func TestHandleGetChannelStatus_NotFound(t *testing.T) {
	_, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodGet, "/api/channels/telegram/status", nil)
	withBearerToken(req, token)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (not found), body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if resp.Success {
		t.Error("success = true, want false when channel not found")
	}
}

// TestChannelCRUDRoutesRegistered verifies that all channel CRUD routes are registered.
func TestChannelCRUDRoutesRegistered(t *testing.T) {
	_, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	testCases := []struct {
		method       string
		path         string
		expectStatus int
		description  string
	}{
		{
			method:       "GET",
			path:         "/api/channels",
			expectStatus: http.StatusOK, // or 403 from non-LAN, but not 404
			description:  "List channels endpoint should be registered",
		},
		{
			method:       "POST",
			path:         "/api/channels",
			expectStatus: http.StatusBadRequest, // invalid JSON, but route exists
			description:  "Create channel endpoint should be registered",
		},
		{
			method:       "PATCH",
			path:         "/api/channels/telegram",
			expectStatus: http.StatusNotFound, // channel doesn't exist, but route exists
			description:  "Update channel endpoint should be registered",
		},
		{
			method:       "DELETE",
			path:         "/api/channels/telegram",
			expectStatus: http.StatusNotFound, // channel doesn't exist, but route exists
			description:  "Delete channel endpoint should be registered",
		},
		{
			method:       "GET",
			path:         "/api/channels/telegram/status",
			expectStatus: http.StatusNotFound, // channel doesn't exist, but route exists
			description:  "Get channel status endpoint should be registered",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			mux.ServeHTTP(rec, req)

			// We expect the route to be registered (not 404)
			// The actual status may vary based on LAN restriction and config state
			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s %s returned 404 (route not registered)", tc.method, tc.path)
			}
		})
	}
}

// TestHandleCreateChannel_MissingFields tests creating a channel with missing required fields.
func TestHandleCreateChannel_MissingFields(t *testing.T) {
	_, h, _, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// Test with empty config (missing required fields)
	createReq := channelCreateRequest{
		ID:     "telegram",
		Config: map[string]any{},
	}

	body, _ := json.Marshal(createReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPost, "/api/channels", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.handleCreateChannel(rec, req)

	// Should succeed - fields are not strictly validated at creation
	// The validation happens when the gateway tries to use the config
	if rec.Code != http.StatusCreated && rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 201 or 400, body=%s", rec.Code, rec.Body.String())
	}
}

// TestHandleUpdateChannel_MultipleChannels tests updating different channel types.
func TestHandleUpdateChannel_MultipleChannels(t *testing.T) {
	configPath, h, token, cleanup := setupChannelTestEnv(t)
	defer cleanup()

	// Test updating discord channel
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.Token = "old-token"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	updateReq := channelUpdateRequest{
		Config: map[string]any{
			"token": "new-discord-token",
		},
	}

	body, _ := json.Marshal(updateReq)
	rec := httptest.NewRecorder()
	req := newLocalRequest(http.MethodPatch, "/api/channels/discord", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	withBearerToken(req, token)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify config was updated
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Channels.Discord.Token != "new-discord-token" {
		t.Errorf("discord token = %s, want new-discord-token", cfg.Channels.Discord.Token)
	}
}
