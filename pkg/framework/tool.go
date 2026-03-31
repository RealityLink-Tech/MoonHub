// MoonHub - Your ready-to-use AI assistant
// Plugin Architecture - Tool Plugin Interface
//
// Copyright (c) 2026 MoonHub contributors

package plugin

import (
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// ToolPlugin extends Plugin with tool-specific functionality.
// Tool plugins extend the agent's capabilities with custom tools.
type ToolPlugin interface {
	Plugin

	// CreateTools returns tool instances this plugin provides
	CreateTools(ctx *RuntimeContext) []tools.Tool

	// IsCore indicates if this is a core tool (always available)
	// Non-core tools can be dynamically promoted/discovered
	IsCore() bool
}
