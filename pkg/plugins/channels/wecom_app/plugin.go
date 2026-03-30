// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - WeCom App Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package wecom_app

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelswecom "github.com/RealityLink-Tech/MoonHub/pkg/channels/wecom"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&WeComAppPlugin{})
}

// WeComAppPlugin implements ChannelPlugin interface for WeCom App
type WeComAppPlugin struct{}

// Metadata returns plugin metadata
func (p *WeComAppPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-wecom_app",
		Name:        "WeCom App",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "WeCom App channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *WeComAppPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *WeComAppPlugin) Validate(cfg *config.Config) error {
	if cfg.Channels.WeComApp.Enabled && cfg.Channels.WeComApp.CorpID == "" {
		return fmt.Errorf("wecom_app corp_id required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *WeComAppPlugin) ChannelPrefix() string {
	return "wecom_app"
}

// IsEnabled checks if the channel is enabled in config
func (p *WeComAppPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.WeComApp.Enabled && cfg.Channels.WeComApp.CorpID != ""
}

// CreateChannel instantiates the channel implementation
func (p *WeComAppPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelswecom.NewWeComAppChannel(cfg.Channels.WeComApp, bus)
}
