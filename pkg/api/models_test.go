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

func TestAddModel_Success(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	body := `{"model_name":"test-model","model":"openai/gpt-4o","api_key":"sk-test1234567890"}`
	req := httptest.NewRequest("POST", "/api/models", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ok", resp["status"])
	assert.NotNil(t, resp["index"])

	// Verify the model was added
	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	found := false
	for _, m := range cfg.ModelList {
		if m.ModelName == "test-model" {
			found = true
			assert.Equal(t, "openai/gpt-4o", m.Model)
			assert.Equal(t, "sk-test1234567890", m.APIKey)
		}
	}
	assert.True(t, found, "model should be in the list")
}

func TestAddModel_DuplicateName(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// First add succeeds
	body := `{"model_name":"dup-model","model":"openai/gpt-4o"}`
	req := httptest.NewRequest("POST", "/api/models", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second add with same name fails
	req2 := httptest.NewRequest("POST", "/api/models", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)

	var resp map[string]string
	json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "already exists")
}

func TestAddModel_ValidationError(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	tests := []struct {
		name string
		body string
	}{
		{"missing model_name", `{"model":"openai/gpt-4o"}`},
		{"missing model", `{"model_name":"test"}`},
		{"invalid json", `{invalid}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/models", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestDeleteModel_Success(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// First add a model
	body := `{"model_name":"to-delete","model":"openai/gpt-4o"}`
	req := httptest.NewRequest("POST", "/api/models", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var addResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &addResp)
	idx := int(addResp["index"].(float64))

	// Delete it
	req2 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/models/%d", idx), nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var delResp map[string]string
	json.Unmarshal(w2.Body.Bytes(), &delResp)
	assert.Equal(t, "ok", delResp["status"])

	// Verify the model is gone
	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	for _, m := range cfg.ModelList {
		assert.NotEqual(t, "to-delete", m.ModelName)
	}
}

func TestDeleteModel_DefaultCleared(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	// Add a model
	body := `{"model_name":"default-model","model":"openai/gpt-4o"}`
	req := httptest.NewRequest("POST", "/api/models", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var addResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &addResp)
	idx := int(addResp["index"].(float64))

	// Set it as default
	setDefBody := `{"model_name":"default-model"}`
	req2 := httptest.NewRequest("POST", "/api/models/default", strings.NewReader(setDefBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Delete it — default should be cleared
	req3 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/models/%d", idx), nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	cfg, err := config.LoadConfig(h.configPath)
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.Agents.Defaults.ModelName)
}

func TestDeleteModel_InvalidIndex(t *testing.T) {
	h, token, cleanup := setupTestModels(t)
	defer cleanup()

	tests := []struct {
		name   string
		index  string
		status int
	}{
		{"negative", "-1", http.StatusBadRequest},
		{"non-numeric", "abc", http.StatusBadRequest},
		{"out of range", "9999", http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/models/%s", tc.index), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Code)
		})
	}
}
