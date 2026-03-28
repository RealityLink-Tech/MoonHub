package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/provisioning"
	"github.com/stretchr/testify/assert"
)

func setupTestModels(t *testing.T) (*Handler, string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	provPath := filepath.Join(tmpDir, "provisioning.json")
	store, err := provisioning.NewJSONConfigStore(provPath)
	assert.NoError(t, err)
	store.Set(provisioning.KeyAuthCode, "123456")
	store.Set(provisioning.KeyDeviceId, "TEST-001")
	store.Save()

	h := NewHandler(configPath, nil, nil)

	// Bind to get a valid token
	bindReq := httptest.NewRequest("POST", "/api/auth/bind", strings.NewReader(`{"code":"123456"}`))
	bindReq.Header.Set("Content-Type", "application/json")
	bindW := httptest.NewRecorder()
	h.ServeHTTP(bindW, bindReq)

	var bindResp bindResponse
	json.Unmarshal(bindW.Body.Bytes(), &bindResp)

	return h, bindResp.Token, func() { os.RemoveAll(tmpDir) }
}

func TestUpdateModel_APIKey(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Update API key for model at index 0
	newAPIKey := "sk-test-new-key-1234567890"
	body := fmt.Sprintf(`{"api_key":"%s"}`, newAPIKey)
	req := httptest.NewRequest("PATCH", "/api/models/0", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp["status"])

	// Verify the change was persisted
	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	assert.Equal(t, newAPIKey, cfg.ModelList[0].APIKey)
}

func TestUpdateModel_EmptyKeyPreserves(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// First, set an API key
	initialKey := "sk-initial-key-1234567890"
	cfg, _ := config.LoadConfig(h.configPath)
	cfg.ModelList[0].APIKey = initialKey
	config.SaveConfig(h.configPath, cfg)

	// Update with empty string - should preserve the existing key
	body := `{"api_key":""}`
	req := httptest.NewRequest("PATCH", "/api/models/0", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the key was preserved
	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	assert.Equal(t, initialKey, cfg.ModelList[0].APIKey)
}

func TestUpdateModel_InvalidIndex(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Try to update a model with an out-of-range index
	body := `{"api_key":"sk-test-key"}`
	req := httptest.NewRequest("PATCH", "/api/models/9999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "out of range")
}

func TestUpdateModel_NegativeIndex(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"api_key":"sk-test-key"}`
	req := httptest.NewRequest("PATCH", "/api/models/-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid model index")
}

func TestUpdateModel_PartialUpdate(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Update only api_base, leaving other fields intact
	newAPIBase := "https://custom.example.com/v1"
	body := fmt.Sprintf(`{"api_base":"%s"}`, newAPIBase)
	req := httptest.NewRequest("PATCH", "/api/models/0", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify only api_base was updated
	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	assert.Equal(t, newAPIBase, cfg.ModelList[0].APIBase)
	assert.Equal(t, "", cfg.ModelList[0].APIKey) // Should still be empty
}

func TestUpdateModel_AllFields(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Update all supported fields
	newRPM := 100
	newThinkingLevel := "high"
	body := fmt.Sprintf(`{
		"api_key": "sk-test-key-1234567890",
		"api_base": "https://custom.example.com/v1",
		"proxy": "http://proxy.example.com:8080",
		"rpm": %d,
		"thinking_level": "%s"
	}`, newRPM, newThinkingLevel)

	req := httptest.NewRequest("PATCH", "/api/models/0", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	m := cfg.ModelList[0]
	assert.Equal(t, "sk-test-key-1234567890", m.APIKey)
	assert.Equal(t, "https://custom.example.com/v1", m.APIBase)
	assert.Equal(t, "http://proxy.example.com:8080", m.Proxy)
	assert.Equal(t, newRPM, m.RPM)
	assert.Equal(t, newThinkingLevel, m.ThinkingLevel)
}

func TestUpdateModel_InvalidJSON(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"api_key": invalid json`
	req := httptest.NewRequest("PATCH", "/api/models/0", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestUpdateModel_NoAuth(t *testing.T) {
	h, _, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"api_key":"sk-test-key"}`
	req := httptest.NewRequest("PATCH", "/api/models/0", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
