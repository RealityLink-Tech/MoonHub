package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

// handleGetConfig returns the complete system configuration with sensitive fields masked.
//
//	GET /api/config
func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		// During initial setup, the config file may not exist yet.
		// Fall back to the default configuration rather than returning an error.
		cfg = config.DefaultConfig()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeConfig(cfg))
}

// handlePatchConfig partially updates the system configuration using JSON Merge Patch (RFC 7396).
// Only the fields present in the request body will be updated; all other fields remain unchanged.
//
//	PATCH /api/config
func (h *Handler) handlePatchConfig(w http.ResponseWriter, r *http.Request) {
	patchBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Validate the patch is valid JSON
	var patch map[string]any
	if err = json.Unmarshal(patchBody, &patch); err != nil {
		WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	// Load existing config and marshal to a map for merging.
	// Fall back to default config if the file doesn't exist yet (initial setup).
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		// Only fall back to defaults if the file doesn't exist yet (initial setup).
		// For corrupted/unreadable files, return an error.
		if !os.IsNotExist(err) {
			WriteJSONError(w, http.StatusInternalServerError, "Failed to load config")
			return
		}
		cfg = config.DefaultConfig()
	}

	existing, err := json.Marshal(cfg)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "Failed to serialize current config")
		return
	}

	var base map[string]any
	if err = json.Unmarshal(existing, &base); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "Failed to parse current config")
		return
	}

	// Recursively merge patch into base
	mergeMap(base, patch)

	// Convert merged map back to Config struct
	merged, err := json.Marshal(base)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "Failed to serialize merged config")
		return
	}

	var newCfg config.Config
	if err := json.Unmarshal(merged, &newCfg); err != nil {
		WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Merged config is invalid: %v", err))
		return
	}

	if errs := validateConfig(&newCfg); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "validation_error",
			"errors": errs,
		})
		return
	}

	if err := config.SaveConfig(h.configPath, &newCfg); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// validateConfig checks the config for common errors before saving.
// Returns a list of human-readable error strings; empty means valid.
func validateConfig(cfg *config.Config) []string {
	var errs []string

	// Validate model_list entries
	if err := cfg.ValidateModelList(); err != nil {
		errs = append(errs, err.Error())
	}

	// Gateway port range
	if cfg.Gateway.Port != 0 && (cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535) {
		errs = append(errs, fmt.Sprintf("gateway.port %d is out of valid range (1-65535)", cfg.Gateway.Port))
	}

	// Telegram: token required when enabled
	if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token == "" {
		errs = append(errs, "channels.telegram.token is required when telegram channel is enabled")
	}

	// Discord: token required when enabled
	if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token == "" {
		errs = append(errs, "channels.discord.token is required when discord channel is enabled")
	}

	return errs
}

// mergeMap recursively merges src into dst (JSON Merge Patch semantics).
// - If a key in src has a null value, it is deleted from dst.
// - If both dst and src have a nested object for the same key, merge recursively.
// - Otherwise the value from src overwrites dst.
func mergeMap(dst, src map[string]any) {
	for key, srcVal := range src {
		if srcVal == nil {
			delete(dst, key)
			continue
		}
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dst[key].(map[string]any)
		if srcIsMap && dstIsMap {
			mergeMap(dstMap, srcMap)
		} else {
			dst[key] = srcVal
		}
	}
}

// sanitizeConfig returns a deep copy of the config with sensitive fields masked.
func sanitizeConfig(cfg *config.Config) map[string]any {
	data, _ := json.Marshal(cfg)
	var raw map[string]any
	json.Unmarshal(data, &raw)

	// Mask model_list API keys (top-level array)
	maskPaths(raw, "model_list", "api_key")
	maskPaths(raw, "model_list", "token")

	// Mask channel tokens/secrets (nested under "channels")
	maskPaths(raw, "channels", "telegram", "token")
	maskPaths(raw, "channels", "discord", "token")
	maskPaths(raw, "channels", "slack", "token")
	maskPaths(raw, "channels", "slack", "app_token")
	maskPaths(raw, "channels", "slack", "bot_token")
	maskPaths(raw, "channels", "feishu", "app_secret")
	maskPaths(raw, "channels", "feishu", "encrypt_key")
	maskPaths(raw, "channels", "dingtalk", "app_secret")
	maskPaths(raw, "channels", "dingtalk", "client_secret")
	maskPaths(raw, "channels", "qq", "app_secret")
	maskPaths(raw, "channels", "wecom", "corp_secret")
	maskPaths(raw, "channels", "wecom", "encoding_aes_key")
	maskPaths(raw, "channels", "wecom_app", "corp_secret")
	maskPaths(raw, "channels", "wecom_aibot", "corp_secret")
	maskPaths(raw, "channels", "line", "channel_secret")
	maskPaths(raw, "channels", "line", "channel_access_token")
	maskPaths(raw, "channels", "irc", "password")
	maskPaths(raw, "channels", "irc", "nickserv_password")
	maskPaths(raw, "channels", "irc", "sasl_password")
	maskPaths(raw, "channels", "whatsapp", "token")
	maskPaths(raw, "channels", "matrix", "access_token")
	maskPaths(raw, "channels", "onebot", "access_token")
	maskPaths(raw, "channels", "moonhub", "token")

	// Mask provider API keys
	maskProviderFields(raw)

	// Mask web tool API keys
	maskWebToolFields(raw)

	// Mask skills tokens
	maskPaths(raw, "tools", "skills", "github", "token")
	maskPaths(raw, "tools", "skills", "registries", "clawhub", "auth_token")

	return raw
}

// maskProviderFields masks all api_key fields under providers.*.
func maskProviderFields(raw map[string]any) {
	providers, ok := raw["providers"]
	if !ok {
		return
	}
	pm, ok := providers.(map[string]any)
	if !ok {
		return
	}
	for _, v := range pm {
		if m, ok := v.(map[string]any); ok {
			maskValue(m, "api_key")
		}
	}
}

// maskWebToolFields masks api_key fields under tools.web.*.
func maskWebToolFields(raw map[string]any) {
	tools, ok := raw["tools"].(map[string]any)
	if !ok {
		return
	}
	web, ok := tools["web"].(map[string]any)
	if !ok {
		return
	}
	for _, v := range web {
		if m, ok := v.(map[string]any); ok {
			maskValue(m, "api_key")
		}
	}
}

// maskPaths navigates into a nested map/array structure and masks a leaf field.
// For intermediate map keys, it descends; for intermediate array keys, it
// iterates every element.
func maskPaths(data map[string]any, path ...string) {
	if len(path) == 0 {
		return
	}

	// Navigate to parent
	current := data
	for i := 0; i < len(path)-1; i++ {
		val, ok := current[path[i]]
		if !ok {
			return
		}
		switch v := val.(type) {
		case map[string]any:
			current = v
		case []any:
			// For arrays (like model_list), apply to each element
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					maskValue(m, path[len(path)-1])
				}
			}
			return
		default:
			return
		}
	}

	maskValue(current, path[len(path)-1])
}

// maskValue replaces a string value in a map with a masked version.
// Values longer than 8 chars show first 3 + "****" + last 4.
// Shorter non-empty values are fully masked as "****".
func maskValue(m map[string]any, key string) {
	val, ok := m[key]
	if !ok {
		return
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return
	}
	if len(s) > 8 {
		m[key] = s[:3] + "****" + s[len(s)-4:]
	} else {
		m[key] = "****"
	}
}
