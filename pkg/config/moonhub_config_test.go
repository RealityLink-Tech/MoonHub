package config

import (
	"encoding/json"
	"testing"
)

func TestMoonHubConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Channels.MoonHub.Enabled {
		t.Error("expected MoonHub disabled by default")
	}
	if cfg.Channels.MoonHub.Port == 0 {
		t.Error("expected non-zero default port")
	}
}

func TestMoonHubConfig_JSONRoundTrip(t *testing.T) {
	original := MoonHubConfig{
		Enabled:   true,
		Port:      18801,
		Token:     "test-token",
		AllowFrom: FlexibleStringSlice{"mh_abcdef1234567890"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded MoonHubConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Enabled != original.Enabled {
		t.Error("Enabled mismatch")
	}
	if decoded.Port != original.Port {
		t.Errorf("Port mismatch: got %d, want %d", decoded.Port, original.Port)
	}
	if decoded.Token != original.Token {
		t.Error("Token mismatch")
	}
}
