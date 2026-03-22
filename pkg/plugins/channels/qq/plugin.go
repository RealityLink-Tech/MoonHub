// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - QQ Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package qq

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsqq "github.com/RealityLink-Tech/MoonHub/pkg/channels/qq"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&QQPlugin{})
}

// QQPlugin implements ChannelPlugin interface for QQ
type QQPlugin struct{}

// Metadata returns plugin metadata
func (p *QQPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-qq",
		Name:        "QQ",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "QQ bot channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *QQPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *QQPlugin) Validate(cfg *config.Config) error {
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *QQPlugin) ChannelPrefix() string {
	return "qq"
}

// IsEnabled checks if the channel is enabled in config
func (p *QQPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.QQ.Enabled
}

// CreateChannel instantiates the channel implementation
func (p *QQPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsqq.NewQQChannel(cfg.Channels.QQ, bus)
}
