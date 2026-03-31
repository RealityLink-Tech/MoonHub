// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Builtin Plugin Index
//
// Copyright (c) 2026 MoonHub contributors

package plugin

// LoadBuiltin ensures all builtin plugins are registered.
// Plugin registration happens via init() when plugin packages are imported.
// The gateway (cmd/moonhub/internal/gateway/helpers.go) imports all plugin packages to trigger registration:
//   - Channel plugins: pkg/plugins/channels/*
//   - Provider plugins: pkg/plugins/providers/*
//   - Tool plugins: pkg/plugins/tools/*
//
// This function is a no-op kept for API compatibility.
func LoadBuiltin() {
	// Plugin imports are in cmd/moonhub/internal/gateway/helpers.go
	// to avoid circular import: plugin -> plugins -> channels -> plugin
}
