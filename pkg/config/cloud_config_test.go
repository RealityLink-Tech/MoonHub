// MoonHub - Your ready-to-use AI assistant
// License: MIT
//
// Copyright (c) 2026 MoonHub contributors

package config

import (
	"encoding/json"
	"testing"
)

func TestCloudConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Cloud.Enabled {
		t.Error("cloud should be disabled by default")
	}
	if cfg.Cloud.DirectoryURL != "" {
		t.Error("directory URL should be empty by default")
	}
	if cfg.Cloud.RelayURL != "" {
		t.Error("relay URL should be empty by default")
	}
	if cfg.Cloud.HeartbeatInterval != 60 {
		t.Errorf("heartbeat interval = %d, want 60", cfg.Cloud.HeartbeatInterval)
	}
	if cfg.Cloud.RegisterOnBoot {
		t.Error("register on boot should be false by default")
	}
}

func TestCloudConfig_JSONRoundTrip(t *testing.T) {
	original := CloudConfig{
		Enabled:           true,
		DirectoryURL:      "https://dir.example.com",
		RelayURL:          "wss://relay.example.com",
		HeartbeatInterval: 30,
		RegisterOnBoot:    true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded CloudConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}
