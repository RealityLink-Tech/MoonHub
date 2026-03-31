// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - MaixCam Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package maixcam

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsmaixcam "github.com/RealityLink-Tech/MoonHub/pkg/channels/maixcam"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&MaixCamPlugin{})
}

// MaixCamPlugin implements ChannelPlugin interface for MaixCam
type MaixCamPlugin struct{}

// Metadata returns plugin metadata
func (p *MaixCamPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-maixcam",
		Name:        "MaixCam",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "MaixCam device channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *MaixCamPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *MaixCamPlugin) Validate(cfg *config.Config) error {
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *MaixCamPlugin) ChannelPrefix() string {
	return "maixcam"
}

// IsEnabled checks if the channel is enabled in config
func (p *MaixCamPlugin) IsEnabled(cfg *config.Config) bool {
	return cfg.Channels.MaixCam.Enabled
}

// CreateChannel instantiates the channel implementation
func (p *MaixCamPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsmaixcam.NewMaixCamChannel(cfg.Channels.MaixCam, bus)
}
