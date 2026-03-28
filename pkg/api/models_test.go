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

func TestSetDefaultModel_Success(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Set default model to "gpt-5.4" which exists in default config
	body := `{"model_name":"gpt-5.4"}`
	req := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "gpt-5.4", resp["default_model"])

	// Verify the change was persisted
	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-5.4", cfg.Agents.Defaults.ModelName)
}

func TestSetDefaultModel_NotFound(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Try to set default to a non-existent model
	body := `{"model_name":"non-existent-model"}`
	req := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "not found in model_list")
}

func TestSetDefaultModel_EmptyName(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"model_name":""}`
	req := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "model_name is required")
}

func TestSetDefaultModel_InvalidJSON(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"model_name": invalid json`
	req := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid JSON")
}

func TestSetDefaultModel_NoAuth(t *testing.T) {
	h, _, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"model_name":"gpt-5.4"}`
	req := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetDefaultModel_MultipleModels(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Test setting to different models in the default config
	models := []string{"glm-4.7", "claude-sonnet-4.6", "deepseek-chat"}

	for _, model := range models {
		body := fmt.Sprintf(`{"model_name":"%s"}`, model)
		req := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the change was persisted
		cfg, err := config.LoadConfig(h.configPath)
		assert.NoError(t, err)
		assert.Equal(t, model, cfg.Agents.Defaults.ModelName)
	}
}
