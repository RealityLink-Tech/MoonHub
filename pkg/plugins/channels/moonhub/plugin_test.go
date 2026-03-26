package moonhub

import (
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	plugin "github.com/RealityLink-Tech/MoonHub/pkg/framework"
)

func TestMoonHubPlugin_Metadata(t *testing.T) {
	p := &MoonHubPlugin{}
	meta := p.Metadata()

	if meta.ID != "moonhub-channel-moonhub" {
		t.Errorf("expected ID moonhub-channel-moonhub, got %s", meta.ID)
	}
	if meta.Name != "MoonHub" {
		t.Errorf("expected Name MoonHub, got %s", meta.Name)
	}
	if meta.Type != plugin.TypeChannel {
		t.Errorf("expected TypeChannel, got %s", meta.Type)
	}
}

func TestMoonHubPlugin_ChannelPrefix(t *testing.T) {
	p := &MoonHubPlugin{}
	if p.ChannelPrefix() != "moonhub" {
		t.Errorf("expected prefix moonhub, got %s", p.ChannelPrefix())
	}
}

func TestMoonHubPlugin_IsEnabled(t *testing.T) {
	p := &MoonHubPlugin{}

	cfg := &config.Config{}
	cfg.Channels.MoonHub.Enabled = true
	cfg.Channels.MoonHub.Token = "test-token"

	if !p.IsEnabled(cfg) {
		t.Error("expected IsEnabled=true when enabled with token")
	}

	cfg.Channels.MoonHub.Token = ""
	if p.IsEnabled(cfg) {
		t.Error("expected IsEnabled=false when token is empty")
	}
}

func TestMoonHubPlugin_Validate(t *testing.T) {
	p := &MoonHubPlugin{}

	cfg := &config.Config{}
	cfg.Channels.MoonHub.Enabled = true
	cfg.Channels.MoonHub.Token = ""

	err := p.Validate(cfg)
	if err == nil {
		t.Error("expected error when enabled without token")
	}

	cfg.Channels.MoonHub.Token = "test-token"
	err = p.Validate(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
