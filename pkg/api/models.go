package api

import (
	"encoding/json"
	"net/http"
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
