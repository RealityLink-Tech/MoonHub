// MoonHub - Ultra-lightweight personal AI agent
// Plugin Architecture - Matrix Channel Plugin
//
// Copyright (c) 2026 MoonHub contributors

package matrix

import (
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/bus"
	"github.com/RealityLink-Tech/MoonHub/pkg/channels"
	channelsmatrix "github.com/RealityLink-Tech/MoonHub/pkg/channels/matrix"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func init() {
	plugin.RegisterPlugin(&MatrixPlugin{})
}

// MatrixPlugin implements ChannelPlugin interface for Matrix
type MatrixPlugin struct{}

// Metadata returns plugin metadata
func (p *MatrixPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		ID:          "moonhub-channel-matrix",
		Name:        "Matrix",
		Type:        plugin.TypeChannel,
		Version:     "2.0.0",
		Description: "Matrix protocol channel integration",
		Priority:    100,
	}
}

// Init initializes the plugin with runtime context
func (p *MatrixPlugin) Init(ctx *plugin.RuntimeContext) error {
	return nil
}

// Validate checks if the plugin can run with current config
func (p *MatrixPlugin) Validate(cfg *config.Config) error {
	mc := cfg.Channels.Matrix
	if mc.Enabled && (mc.Homeserver == "" || mc.UserID == "" || mc.AccessToken == "") {
		return fmt.Errorf("matrix homeserver, user_id and and access_token required when enabled")
	}
	return nil
}

// ChannelPrefix returns the userId prefix this channel owns
func (p *MatrixPlugin) ChannelPrefix() string {
	return "matrix"
}

// IsEnabled checks if the channel is enabled in config
func (p *MatrixPlugin) IsEnabled(cfg *config.Config) bool {
	mc := cfg.Channels.Matrix
	return mc.Enabled && mc.Homeserver != "" && mc.UserID != "" && mc.AccessToken != ""
}

// CreateChannel instantiates the channel implementation
func (p *MatrixPlugin) CreateChannel(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
	return channelsmatrix.NewMatrixChannel(cfg.Channels.Matrix, bus)
}
