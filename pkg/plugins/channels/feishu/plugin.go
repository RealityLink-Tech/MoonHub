// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Feishu Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package feishu

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsfeishu "github.com/RealityLink-Tech/MoonHub/pkg/channels/feishu"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&FeishuPlugin{})
}

// FeishuPlugin implements ChannelPlugin interface for Feishu
type FeishuPlugin struct{}

// Metadata returns plugin metadata
func (p *FeishuPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-feishu",
		Name:        "Feishu",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "Feishu/Lark bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *FeishuPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *FeishuPlugin) Validate(cfg *config.Config) error {
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *FeishuPlugin) ChannelPrefix() string {
	return "feishu"
}

// IsEnabled checks if the channel is enabled in config
func (p *FeishuPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.Feishu.Enabled
}

// CreateChannel instantiates the channel implementation
func (p *FeishuPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsfeishu.NewFeishuChannel(cfg.Channels.Feishu, bus)
}
