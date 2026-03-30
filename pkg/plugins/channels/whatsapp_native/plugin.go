// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - WhatsApp Native Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package whatsapp_native

import (
	"path/filepath"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelswhatsappnative "github.com/RealityLink-Tech/MoonHub/pkg/channels/whatsapp_native"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&WhatsAppNativePlugin{})
}

// WhatsAppNativePlugin implements ChannelPlugin interface for WhatsApp Native
type WhatsAppNativePlugin struct{}

// Metadata returns plugin metadata
func (p *WhatsAppNativePlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-whatsapp_native",
		Name:        "WhatsApp Native",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "WhatsApp native channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *WhatsAppNativePlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *WhatsAppNativePlugin) Validate(cfg *config.Config) error {
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *WhatsAppNativePlugin) ChannelPrefix() string {
	return "whatsapp"
}

// IsEnabled checks if the channel is enabled in config
func (p *WhatsAppNativePlugin) IsEnabled(cfg *config.Config) bool {
	wa := cfg.Channels.WhatsApp
	return wa.Enabled && wa.UseNative
}

// CreateChannel instantiates the channel implementation
func (p *WhatsAppNativePlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	waCfg := cfg.Channels.WhatsApp
	storePath := waCfg.SessionStorePath
	if storePath == "" {
		storePath = filepath.Join(cfg.WorkspacePath(), "whatsapp")
	}
	return channelswhatsappnative.NewWhatsAppNativeChannel(waCfg, bus, storePath)
}
