package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

// registerChannelCRUDRoutes binds channel CRUD endpoints to the ServeMux.
func (h *Handler) registerChannelCRUDRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/channels", h.handleListChannels)
	mux.HandleFunc("POST /api/channels", h.handleCreateChannel)
	mux.HandleFunc("PATCH /api/channels/{id}", h.handleUpdateChannel)
	mux.HandleFunc("DELETE /api/channels/{id}", h.handleDeleteChannel)
	mux.HandleFunc("GET /api/channels/{id}/status", h.handleGetChannelStatus)
}

// channelInstance represents a configured channel instance with its status.
type channelInstance struct {
	ID      string         `json:"id"`      // Channel type (e.g., "telegram", "discord")
	Name    string         `json:"name"`    // Display name from catalog
	Enabled bool           `json:"enabled"` // Whether the channel is enabled
	Config  map[string]any `json:"config"`  // Channel configuration (non-sensitive fields only)
	Variant string         `json:"variant"` // Optional variant (e.g., "bridge", "native")
	Status  map[string]any `json:"status"`  // Runtime status (enabled, running)
}

// channelCreateRequest represents a request to create a new channel configuration.
type channelCreateRequest struct {
	ID     string         `json:"id"`     // Channel type (must match catalog)
	Type   string         `json:"type"`   // Alias for id (PWA and other clients)
	Name   string         `json:"name"`   // Optional; catalog supplies display names
	Config map[string]any `json:"config"` // Channel configuration
}

// channelUpdateRequest represents a request to update a channel configuration.
type channelUpdateRequest struct {
	Config map[string]any `json:"config"` // Channel configuration fields to update
}

// handleListChannels returns all configured channel instances with their status.
//
//	GET /api/channels
func (h *Handler) handleListChannels(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	instances := h.buildChannelInstances(cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    instances,
	})
}

// handleCreateChannel creates a new channel configuration.
//
//	POST /api/channels
func (h *Handler) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	var req channelCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	channelID := strings.TrimSpace(req.ID)
	if channelID == "" {
		channelID = strings.TrimSpace(req.Type)
	}
	if channelID == "" {
		writeJSONError(w, http.StatusBadRequest, "Channel id or type is required")
		return
	}

	// Validate channel type against catalog
	catalogItem, ok := h.findCatalogItem(channelID)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Unknown channel type: %s", channelID))
		return
	}

	// Load current config
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	// Check if channel is already configured
	if h.isChannelEnabled(cfg, channelID) {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Channel %s is already configured", channelID))
		return
	}

	// Apply the configuration
	if err := h.applyChannelConfig(cfg, channelID, catalogItem.ConfigKey, req.Config, true); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Save config
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":      channelID,
			"message": "Channel created successfully",
		},
	})
}

// handleUpdateChannel updates an existing channel configuration.
//
//	PATCH /api/channels/{id}
func (h *Handler) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	channelID := r.PathValue("id")
	if channelID == "" {
		writeJSONError(w, http.StatusBadRequest, "Channel ID is required")
		return
	}

	// Validate channel type against catalog
	catalogItem, ok := h.findCatalogItem(channelID)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Unknown channel type: %s", channelID))
		return
	}

	var req channelUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Load current config
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	// Check if channel exists
	if !h.isChannelEnabled(cfg, channelID) {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Channel %s is not configured", channelID))
		return
	}

	// Apply the configuration update
	if err := h.applyChannelConfig(cfg, channelID, catalogItem.ConfigKey, req.Config, false); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Save config
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":      channelID,
			"message": "Channel updated successfully",
		},
	})
}

// handleDeleteChannel deletes a channel configuration.
//
//	DELETE /api/channels/{id}
func (h *Handler) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	channelID := r.PathValue("id")
	if channelID == "" {
		writeJSONError(w, http.StatusBadRequest, "Channel ID is required")
		return
	}

	// Validate channel type against catalog
	_, ok := h.findCatalogItem(channelID)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Unknown channel type: %s", channelID))
		return
	}

	// Load current config
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	// Check if channel exists
	if !h.isChannelEnabled(cfg, channelID) {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Channel %s is not configured", channelID))
		return
	}

	// Disable the channel by setting enabled to false
	if err := h.disableChannel(cfg, channelID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Save config
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":      channelID,
			"message": "Channel deleted successfully",
		},
	})
}

// handleGetChannelStatus returns the runtime status of a single channel.
//
//	GET /api/channels/{id}/status
func (h *Handler) handleGetChannelStatus(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	channelID := r.PathValue("id")
	if channelID == "" {
		writeJSONError(w, http.StatusBadRequest, "Channel ID is required")
		return
	}

	// Validate channel type against catalog
	_, ok := h.findCatalogItem(channelID)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Unknown channel type: %s", channelID))
		return
	}

	// Load current config
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	// Check if channel exists
	if !h.isChannelEnabled(cfg, channelID) {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Channel %s is not configured", channelID))
		return
	}

	// Build status response
	status := map[string]any{
		"enabled": true,
		"running": false, // Web backend doesn't have access to runtime channel manager
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":     channelID,
			"status": status,
		},
	})
}

// buildChannelInstances builds a list of channel instances from the config.
func (h *Handler) buildChannelInstances(cfg *config.Config) []channelInstance {
	var instances []channelInstance

	for _, item := range channelCatalog {
		enabled := h.isChannelEnabled(cfg, item.Name)
		if !enabled {
			continue
		}

		instance := channelInstance{
			ID:      item.Name,
			Name:    item.Name,
			Enabled: true,
			Config:  h.extractChannelConfig(cfg, item),
			Variant: item.Variant,
			Status: map[string]any{
				"enabled": true,
				"running": false, // Web backend doesn't have runtime access
			},
		}

		instances = append(instances, instance)
	}

	return instances
}

// findCatalogItem finds a catalog item by channel name.
func (h *Handler) findCatalogItem(name string) (channelCatalogItem, bool) {
	for _, item := range channelCatalog {
		if item.Name == name {
			return item, true
		}
	}
	return channelCatalogItem{}, false
}

// isChannelEnabled checks if a channel is enabled in the config.
func (h *Handler) isChannelEnabled(cfg *config.Config, name string) bool {
	switch name {
	case "telegram":
		return cfg.Channels.Telegram.Enabled
	case "discord":
		return cfg.Channels.Discord.Enabled
	case "slack":
		return cfg.Channels.Slack.Enabled
	case "feishu":
		return cfg.Channels.Feishu.Enabled
	case "dingtalk":
		return cfg.Channels.DingTalk.Enabled
	case "line":
		return cfg.Channels.LINE.Enabled
	case "qq":
		return cfg.Channels.QQ.Enabled
	case "onebot":
		return cfg.Channels.OneBot.Enabled
	case "wecom":
		return cfg.Channels.WeCom.Enabled
	case "wecom_app":
		return cfg.Channels.WeComApp.Enabled
	case "wecom_aibot":
		return cfg.Channels.WeComAIBot.Enabled
	case "whatsapp", "whatsapp_bridge":
		return cfg.Channels.WhatsApp.Enabled && !cfg.Channels.WhatsApp.UseNative
	case "whatsapp_native":
		return cfg.Channels.WhatsApp.Enabled && cfg.Channels.WhatsApp.UseNative
	case "moonhub":
		return cfg.Channels.MoonHub.Enabled
	case "maixcam":
		return cfg.Channels.MaixCam.Enabled
	case "matrix":
		return cfg.Channels.Matrix.Enabled
	case "irc":
		return cfg.Channels.IRC.Enabled
	default:
		return false
	}
}

// extractChannelConfig extracts the non-sensitive configuration for a channel.
func (h *Handler) extractChannelConfig(cfg *config.Config, item channelCatalogItem) map[string]any {
	configMap := make(map[string]any)

	switch item.Name {
	case "telegram":
		if cfg.Channels.Telegram.Enabled {
			configMap["base_url"] = cfg.Channels.Telegram.BaseURL
			configMap["proxy"] = cfg.Channels.Telegram.Proxy
			configMap["allow_from"] = cfg.Channels.Telegram.AllowFrom
		}
	case "discord":
		if cfg.Channels.Discord.Enabled {
			configMap["proxy"] = cfg.Channels.Discord.Proxy
			configMap["allow_from"] = cfg.Channels.Discord.AllowFrom
		}
	case "slack":
		if cfg.Channels.Slack.Enabled {
			configMap["allow_from"] = cfg.Channels.Slack.AllowFrom
		}
	case "feishu":
		if cfg.Channels.Feishu.Enabled {
			configMap["app_id"] = cfg.Channels.Feishu.AppID
			configMap["allow_from"] = cfg.Channels.Feishu.AllowFrom
		}
	case "dingtalk":
		if cfg.Channels.DingTalk.Enabled {
			configMap["client_id"] = cfg.Channels.DingTalk.ClientID
			configMap["allow_from"] = cfg.Channels.DingTalk.AllowFrom
		}
	case "line":
		if cfg.Channels.LINE.Enabled {
			configMap["webhook_host"] = cfg.Channels.LINE.WebhookHost
			configMap["webhook_port"] = cfg.Channels.LINE.WebhookPort
			configMap["allow_from"] = cfg.Channels.LINE.AllowFrom
		}
	case "qq":
		if cfg.Channels.QQ.Enabled {
			configMap["app_id"] = cfg.Channels.QQ.AppID
			configMap["allow_from"] = cfg.Channels.QQ.AllowFrom
		}
	case "onebot":
		if cfg.Channels.OneBot.Enabled {
			configMap["ws_url"] = cfg.Channels.OneBot.WSUrl
			configMap["allow_from"] = cfg.Channels.OneBot.AllowFrom
		}
	case "wecom":
		if cfg.Channels.WeCom.Enabled {
			configMap["webhook_url"] = cfg.Channels.WeCom.WebhookURL
			configMap["webhook_host"] = cfg.Channels.WeCom.WebhookHost
			configMap["webhook_port"] = cfg.Channels.WeCom.WebhookPort
			configMap["allow_from"] = cfg.Channels.WeCom.AllowFrom
		}
	case "wecom_app":
		if cfg.Channels.WeComApp.Enabled {
			configMap["corp_id"] = cfg.Channels.WeComApp.CorpID
			configMap["agent_id"] = cfg.Channels.WeComApp.AgentID
			configMap["webhook_host"] = cfg.Channels.WeComApp.WebhookHost
			configMap["webhook_port"] = cfg.Channels.WeComApp.WebhookPort
			configMap["allow_from"] = cfg.Channels.WeComApp.AllowFrom
		}
	case "wecom_aibot":
		if cfg.Channels.WeComAIBot.Enabled {
			configMap["allow_from"] = cfg.Channels.WeComAIBot.AllowFrom
		}
	case "whatsapp", "whatsapp_bridge", "whatsapp_native":
		if cfg.Channels.WhatsApp.Enabled {
			configMap["use_native"] = cfg.Channels.WhatsApp.UseNative
			configMap["bridge_url"] = cfg.Channels.WhatsApp.BridgeURL
			configMap["allow_from"] = cfg.Channels.WhatsApp.AllowFrom
		}
	case "moonhub":
		if cfg.Channels.MoonHub.Enabled {
			configMap["port"] = cfg.Channels.MoonHub.Port
			configMap["allow_from"] = cfg.Channels.MoonHub.AllowFrom
		}
	case "maixcam":
		if cfg.Channels.MaixCam.Enabled {
			configMap["host"] = cfg.Channels.MaixCam.Host
			configMap["port"] = cfg.Channels.MaixCam.Port
			configMap["allow_from"] = cfg.Channels.MaixCam.AllowFrom
		}
	case "matrix":
		if cfg.Channels.Matrix.Enabled {
			configMap["homeserver"] = cfg.Channels.Matrix.Homeserver
			configMap["user_id"] = cfg.Channels.Matrix.UserID
			configMap["allow_from"] = cfg.Channels.Matrix.AllowFrom
		}
	case "irc":
		if cfg.Channels.IRC.Enabled {
			configMap["server"] = cfg.Channels.IRC.Server
			configMap["tls"] = cfg.Channels.IRC.TLS
			configMap["nick"] = cfg.Channels.IRC.Nick
			configMap["allow_from"] = cfg.Channels.IRC.AllowFrom
		}
	}

	return configMap
}

// applyChannelConfig applies configuration to a channel.
func (h *Handler) applyChannelConfig(cfg *config.Config, channelID, configKey string, configData map[string]any, create bool) error {
	switch channelID {
	case "telegram":
		return h.applyTelegramConfig(cfg, configData, create)
	case "discord":
		return h.applyDiscordConfig(cfg, configData, create)
	case "slack":
		return h.applySlackConfig(cfg, configData, create)
	case "feishu":
		return h.applyFeishuConfig(cfg, configData, create)
	case "dingtalk":
		return h.applyDingTalkConfig(cfg, configData, create)
	case "line":
		return h.applyLINEConfig(cfg, configData, create)
	case "qq":
		return h.applyQQConfig(cfg, configData, create)
	case "onebot":
		return h.applyOneBotConfig(cfg, configData, create)
	case "wecom":
		return h.applyWeComConfig(cfg, configData, create)
	case "wecom_app":
		return h.applyWeComAppConfig(cfg, configData, create)
	case "wecom_aibot":
		return h.applyWeComAIBotConfig(cfg, configData, create)
	case "whatsapp", "whatsapp_bridge", "whatsapp_native":
		return h.applyWhatsAppConfig(cfg, configData, create, channelID)
	case "moonhub":
		return h.applyMoonHubConfig(cfg, configData, create)
	case "maixcam":
		return h.applyMaixCamConfig(cfg, configData, create)
	case "matrix":
		return h.applyMatrixConfig(cfg, configData, create)
	case "irc":
		return h.applyIRCConfig(cfg, configData, create)
	default:
		return fmt.Errorf("unsupported channel type: %s", channelID)
	}
}

// disableChannel disables a channel by setting enabled to false.
func (h *Handler) disableChannel(cfg *config.Config, channelID string) error {
	switch channelID {
	case "telegram":
		cfg.Channels.Telegram.Enabled = false
	case "discord":
		cfg.Channels.Discord.Enabled = false
	case "slack":
		cfg.Channels.Slack.Enabled = false
	case "feishu":
		cfg.Channels.Feishu.Enabled = false
	case "dingtalk":
		cfg.Channels.DingTalk.Enabled = false
	case "line":
		cfg.Channels.LINE.Enabled = false
	case "qq":
		cfg.Channels.QQ.Enabled = false
	case "onebot":
		cfg.Channels.OneBot.Enabled = false
	case "wecom":
		cfg.Channels.WeCom.Enabled = false
	case "wecom_app":
		cfg.Channels.WeComApp.Enabled = false
	case "wecom_aibot":
		cfg.Channels.WeComAIBot.Enabled = false
	case "whatsapp", "whatsapp_bridge", "whatsapp_native":
		cfg.Channels.WhatsApp.Enabled = false
	case "moonhub":
		cfg.Channels.MoonHub.Enabled = false
	case "maixcam":
		cfg.Channels.MaixCam.Enabled = false
	case "matrix":
		cfg.Channels.Matrix.Enabled = false
	case "irc":
		cfg.Channels.IRC.Enabled = false
	default:
		return fmt.Errorf("unsupported channel type: %s", channelID)
	}
	return nil
}

// Channel-specific config appliers
func (h *Handler) applyTelegramConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.Telegram.Enabled = true
	}
	if token, ok := data["token"].(string); ok {
		cfg.Channels.Telegram.Token = token
	}
	if baseURL, ok := data["base_url"].(string); ok {
		cfg.Channels.Telegram.BaseURL = baseURL
	}
	if proxy, ok := data["proxy"].(string); ok {
		cfg.Channels.Telegram.Proxy = proxy
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.Telegram.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyDiscordConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.Discord.Enabled = true
	}
	if token, ok := data["token"].(string); ok {
		cfg.Channels.Discord.Token = token
	}
	if proxy, ok := data["proxy"].(string); ok {
		cfg.Channels.Discord.Proxy = proxy
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.Discord.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applySlackConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.Slack.Enabled = true
	}
	if botToken, ok := data["bot_token"].(string); ok {
		cfg.Channels.Slack.BotToken = botToken
	}
	if appToken, ok := data["app_token"].(string); ok {
		cfg.Channels.Slack.AppToken = appToken
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.Slack.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyFeishuConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.Feishu.Enabled = true
	}
	if appID, ok := data["app_id"].(string); ok {
		cfg.Channels.Feishu.AppID = appID
	}
	if appSecret, ok := data["app_secret"].(string); ok {
		cfg.Channels.Feishu.AppSecret = appSecret
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.Feishu.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyDingTalkConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.DingTalk.Enabled = true
	}
	if clientID, ok := data["client_id"].(string); ok {
		cfg.Channels.DingTalk.ClientID = clientID
	}
	if clientSecret, ok := data["client_secret"].(string); ok {
		cfg.Channels.DingTalk.ClientSecret = clientSecret
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.DingTalk.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyLINEConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.LINE.Enabled = true
	}
	if channelSecret, ok := data["channel_secret"].(string); ok {
		cfg.Channels.LINE.ChannelSecret = channelSecret
	}
	if channelAccessToken, ok := data["channel_access_token"].(string); ok {
		cfg.Channels.LINE.ChannelAccessToken = channelAccessToken
	}
	if webhookHost, ok := data["webhook_host"].(string); ok {
		cfg.Channels.LINE.WebhookHost = webhookHost
	}
	if webhookPort, ok := data["webhook_port"].(float64); ok {
		cfg.Channels.LINE.WebhookPort = int(webhookPort)
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.LINE.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyQQConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.QQ.Enabled = true
	}
	if appID, ok := data["app_id"].(string); ok {
		cfg.Channels.QQ.AppID = appID
	}
	if appSecret, ok := data["app_secret"].(string); ok {
		cfg.Channels.QQ.AppSecret = appSecret
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.QQ.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyOneBotConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.OneBot.Enabled = true
	}
	if wsURL, ok := data["ws_url"].(string); ok {
		cfg.Channels.OneBot.WSUrl = wsURL
	}
	if accessToken, ok := data["access_token"].(string); ok {
		cfg.Channels.OneBot.AccessToken = accessToken
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.OneBot.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyWeComConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.WeCom.Enabled = true
	}
	if token, ok := data["token"].(string); ok {
		cfg.Channels.WeCom.Token = token
	}
	if webhookURL, ok := data["webhook_url"].(string); ok {
		cfg.Channels.WeCom.WebhookURL = webhookURL
	}
	if webhookHost, ok := data["webhook_host"].(string); ok {
		cfg.Channels.WeCom.WebhookHost = webhookHost
	}
	if webhookPort, ok := data["webhook_port"].(float64); ok {
		cfg.Channels.WeCom.WebhookPort = int(webhookPort)
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.WeCom.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyWeComAppConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.WeComApp.Enabled = true
	}
	if corpID, ok := data["corp_id"].(string); ok {
		cfg.Channels.WeComApp.CorpID = corpID
	}
	if corpSecret, ok := data["corp_secret"].(string); ok {
		cfg.Channels.WeComApp.CorpSecret = corpSecret
	}
	if agentID, ok := data["agent_id"].(float64); ok {
		cfg.Channels.WeComApp.AgentID = int64(agentID)
	}
	if webhookHost, ok := data["webhook_host"].(string); ok {
		cfg.Channels.WeComApp.WebhookHost = webhookHost
	}
	if webhookPort, ok := data["webhook_port"].(float64); ok {
		cfg.Channels.WeComApp.WebhookPort = int(webhookPort)
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.WeComApp.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyWeComAIBotConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.WeComAIBot.Enabled = true
	}
	if token, ok := data["token"].(string); ok {
		cfg.Channels.WeComAIBot.Token = token
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.WeComAIBot.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyWhatsAppConfig(cfg *config.Config, data map[string]any, create bool, channelID string) error {
	if create {
		cfg.Channels.WhatsApp.Enabled = true
	}
	if channelID == "whatsapp_native" {
		cfg.Channels.WhatsApp.UseNative = true
	} else if channelID == "whatsapp_bridge" {
		cfg.Channels.WhatsApp.UseNative = false
	}
	if useNative, ok := data["use_native"].(bool); ok {
		cfg.Channels.WhatsApp.UseNative = useNative
	}
	if bridgeURL, ok := data["bridge_url"].(string); ok {
		cfg.Channels.WhatsApp.BridgeURL = bridgeURL
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.WhatsApp.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyMoonHubConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.MoonHub.Enabled = true
	}
	if port, ok := data["port"].(float64); ok {
		cfg.Channels.MoonHub.Port = int(port)
	}
	if token, ok := data["token"].(string); ok {
		cfg.Channels.MoonHub.Token = token
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.MoonHub.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyMaixCamConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.MaixCam.Enabled = true
	}
	if host, ok := data["host"].(string); ok {
		cfg.Channels.MaixCam.Host = host
	}
	if port, ok := data["port"].(float64); ok {
		cfg.Channels.MaixCam.Port = int(port)
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.MaixCam.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyMatrixConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.Matrix.Enabled = true
	}
	if homeserver, ok := data["homeserver"].(string); ok {
		cfg.Channels.Matrix.Homeserver = homeserver
	}
	if userID, ok := data["user_id"].(string); ok {
		cfg.Channels.Matrix.UserID = userID
	}
	if accessToken, ok := data["access_token"].(string); ok {
		cfg.Channels.Matrix.AccessToken = accessToken
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.Matrix.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

func (h *Handler) applyIRCConfig(cfg *config.Config, data map[string]any, create bool) error {
	if create {
		cfg.Channels.IRC.Enabled = true
	}
	if server, ok := data["server"].(string); ok {
		cfg.Channels.IRC.Server = server
	}
	if tls, ok := data["tls"].(bool); ok {
		cfg.Channels.IRC.TLS = tls
	}
	if nick, ok := data["nick"].(string); ok {
		cfg.Channels.IRC.Nick = nick
	}
	if allowFrom, ok := data["allow_from"].([]any); ok {
		cfg.Channels.IRC.AllowFrom = flexibleSliceFromAny(allowFrom)
	}
	return nil
}

// Helper functions
func flexibleSliceFromAny(data []any) config.FlexibleStringSlice {
	result := make(config.FlexibleStringSlice, 0, len(data))
	for _, v := range data {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
