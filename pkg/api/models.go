package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

// modelResponse is the JSON structure returned for each model in the list.
// All ModelConfig fields are included so the frontend can display and edit them.
type modelResponse struct {
	Index      int    `json:"index"`
	ModelName  string `json:"model_name"`
	Model      string `json:"model"`
	APIBase    string `json:"api_base,omitempty"`
	APIKey     string `json:"api_key"`
	Proxy      string `json:"proxy,omitempty"`
	AuthMethod string `json:"auth_method,omitempty"`
	// Advanced fields
	ConnectMode    string `json:"connect_mode,omitempty"`
	Workspace      string `json:"workspace,omitempty"`
	RPM            int    `json:"rpm,omitempty"`
	MaxTokensField string `json:"max_tokens_field,omitempty"`
	RequestTimeout int    `json:"request_timeout,omitempty"`
	ThinkingLevel  string `json:"thinking_level,omitempty"`
	// Meta
	Configured bool `json:"configured"`
	IsDefault  bool `json:"is_default"`
}

// handleListModels returns all model_list entries with masked API keys.
//
//	GET /api/models
func (h *Handler) handleListModels(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	defaultModel := cfg.Agents.Defaults.GetModelName()

	models := make([]modelResponse, 0, len(cfg.ModelList))
	for i, m := range cfg.ModelList {
		models = append(models, modelResponse{
			Index:          i,
			ModelName:      m.ModelName,
			Model:          m.Model,
			APIBase:        m.APIBase,
			APIKey:         maskAPIKey(m.APIKey),
			Proxy:          m.Proxy,
			AuthMethod:     m.AuthMethod,
			ConnectMode:    m.ConnectMode,
			Workspace:      m.Workspace,
			RPM:            m.RPM,
			MaxTokensField: m.MaxTokensField,
			RequestTimeout: m.RequestTimeout,
			ThinkingLevel:  m.ThinkingLevel,
			Configured:     isModelConfigured(m),
			IsDefault:      m.ModelName == defaultModel,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"models":        models,
		"total":         len(models),
		"default_model": defaultModel,
	})
}

// isModelConfigured reports whether a model has the necessary configuration
// to be used (has an API key, or uses OAuth/token auth, or is a local model).
// This is a lightweight check without runtime probing.
func isModelConfigured(m config.ModelConfig) bool {
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	apiKey := strings.TrimSpace(m.APIKey)

	// OAuth or token-based auth methods are considered configured
	if authMethod == "oauth" || authMethod == "token" {
		return true
	}

	// Local auth method doesn't need an API key
	if authMethod == "local" {
		return true
	}

	// CLI-based providers (claude-cli, codex-cli, github-copilot) don't need an API key
	protocol := modelProtocol(m.Model)
	switch protocol {
	case "claude-cli", "claudecli", "codex-cli", "codexcli", "github-copilot", "copilot":
		return true
	}

	// For all other providers, having an API key means configured
	return apiKey != ""
}

// modelProtocol extracts the protocol prefix from a model string.
// e.g., "openai/gpt-4" -> "openai", "gpt-4" -> "openai"
func modelProtocol(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	protocol, _, found := strings.Cut(model, "/")
	if !found {
		return "openai"
	}
	return protocol
}

// maskAPIKey returns a masked version of an API key for safe display.
// Keys longer than 8 chars show prefix + last 4 chars: "sk-****abcd"
// Shorter keys are fully masked as "****".
// Empty keys return empty string.
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	// Show first 3 chars and last 4 chars
	return key[:3] + "****" + key[len(key)-4:]
}

// handleUpdateModel updates a single model entry in model_list by index.
// Only the fields present in the request body are updated.
// If api_key is empty string, the existing key is preserved (allows updating other fields without re-entering key).
//
//	PATCH /api/models/{index}
func (h *Handler) handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	indexStr := r.PathValue("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid model index")
		return
	}

	var patch struct {
		APIKey        *string `json:"api_key"`
		APIBase       *string `json:"api_base"`
		Proxy         *string `json:"proxy"`
		RPM           *int    `json:"rpm"`
		ThinkingLevel *string `json:"thinking_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
			return
		}
		cfg = config.DefaultConfig()
	}

	if index >= len(cfg.ModelList) {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Model index %d out of range", index))
		return
	}

	m := &cfg.ModelList[index]

	if patch.APIKey != nil {
		if *patch.APIKey != "" {
			m.APIKey = *patch.APIKey
		}
		// Empty string → preserve existing key (no-op)
	}
	if patch.APIBase != nil {
		m.APIBase = *patch.APIBase
	}
	if patch.Proxy != nil {
		m.Proxy = *patch.Proxy
	}
	if patch.RPM != nil {
		m.RPM = *patch.RPM
	}
	if patch.ThinkingLevel != nil {
		m.ThinkingLevel = *patch.ThinkingLevel
	}

	if err := cfg.ValidateModelList(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleSetDefaultModel sets the default model by model_name.
//
//	POST /api/models/default
//	Body: {"model_name": "gpt-4o"}
func (h *Handler) handleSetDefaultModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelName string `json:"model_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if req.ModelName == "" {
		writeJSONError(w, http.StatusBadRequest, "model_name is required")
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
			return
		}
		cfg = config.DefaultConfig()
	}

	// Validate model_name exists in model_list
	found := false
	for _, m := range cfg.ModelList {
		if m.ModelName == req.ModelName {
			found = true
			break
		}
	}
	if !found {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Model %q not found in model_list", req.ModelName))
		return
	}

	cfg.Agents.Defaults.ModelName = req.ModelName

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":        "ok",
		"default_model": req.ModelName,
	})
}

// handleAddModel appends a new model configuration entry.
//
//	POST /api/models
func (h *Handler) handleAddModel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var mc config.ModelConfig
	if err = json.Unmarshal(body, &mc); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err = mc.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
			return
		}
		cfg = config.DefaultConfig()
	}

	// Check for duplicate model_name
	for _, existing := range cfg.ModelList {
		if existing.ModelName == mc.ModelName {
			writeJSONError(w, http.StatusConflict, fmt.Sprintf("Model %q already exists", mc.ModelName))
			return
		}
	}

	cfg.ModelList = append(cfg.ModelList, mc)

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"index":  len(cfg.ModelList) - 1,
	})
}

// handleDeleteModel removes a model configuration entry at the given index.
//
//	DELETE /api/models/{index}
func (h *Handler) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	indexStr := r.PathValue("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid model index")
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
			return
		}
		cfg = config.DefaultConfig()
	}

	if index >= len(cfg.ModelList) {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Model index %d out of range", index))
		return
	}

	deletedModelName := cfg.ModelList[index].ModelName

	cfg.ModelList = append(cfg.ModelList[:index], cfg.ModelList[index+1:]...)

	// If the deleted model was the default, clear it
	if cfg.Agents.Defaults.ModelName == deletedModelName {
		cfg.Agents.Defaults.ModelName = ""
	}
	if cfg.Agents.Defaults.Model == deletedModelName {
		cfg.Agents.Defaults.Model = ""
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
