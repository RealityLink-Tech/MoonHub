package mdns

import (
	"context"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	config := ServerConfig{
		DeviceID: "YSHU-2026-ABC",
		Name:     "Test Device",
		Version:  "1.0.0",
		Port:     18800,
	}

	server := NewServer(config)
	if server == nil {
		t.Fatal("expected server instance")
	}

	if server.config.Port != 18800 {
		t.Errorf("expected port 18800, got %d", server.config.Port)
	}
}

func TestNewServerDefaultPort(t *testing.T) {
	config := ServerConfig{
		DeviceID: "YSHU-2026-ABC",
		Name:     "Test Device",
		Version:  "1.0.0",
	}

	server := NewServer(config)
	if server.config.Port != DefaultPort {
		t.Errorf("expected default port %d, got %d", DefaultPort, server.config.Port)
	}
}

func TestSanitizeHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"moonhub", "moonhub"},
		{"MoonHub", "MoonHub"},
		{"moon-hub-123", "moon-hub-123"},
		{"moon_hub!", "moonhub"},
		{"test.device.name", "testdevicename"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeHostname(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeHostname(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseTXTRecords(t *testing.T) {
	txt := []string{
		"id=YSHU-2026-ABC",
		"name=Test Device",
		"version=1.0.0",
		"port=18800",
	}

	result := parseTXTRecords(txt)

	if result["id"] != "YSHU-2026-ABC" {
		t.Errorf("expected id YSHU-2026-ABC, got %s", result["id"])
	}
	if result["name"] != "Test Device" {
		t.Errorf("expected name Test Device, got %s", result["name"])
	}
	if result["version"] != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result["version"])
	}
	if result["port"] != "18800" {
		t.Errorf("expected port 18800, got %s", result["port"])
	}
}

func TestServiceTXTRecords(t *testing.T) {
	config := ServerConfig{
		DeviceID: "YSHU-2026-ABC",
		Name:     "Test Device",
		Version:  "1.0.0",
		Port:     18800,
	}

	server := NewServer(config)
	records := server.serviceTXTRecords()

	if len(records) != 4 {
		t.Errorf("expected 4 TXT records, got %d", len(records))
	}
}

func TestServerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := ServerConfig{
		DeviceID: "YSHU-2026-TEST",
		Name:     "Test Device",
		Version:  "1.0.0",
		Port:     18800,
	}

	server := NewServer(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	if !server.running {
		t.Error("expected server to be running")
	}

	// Stop the server
	server.Stop()

	if server.running {
		t.Error("expected server to be stopped")
	}
}
