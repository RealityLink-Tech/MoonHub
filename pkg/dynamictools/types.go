package dynamictools

import "context"

// DynamicTool represents an AI-generated dynamic tool.
type DynamicTool struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Category     string         `json:"category"`
	ChatSchema   GeneratedComponent `json:"chat_schema"`
	SpaceSchema  GeneratedComponent `json:"space_schema"`
	Engine       string         `json:"engine"`        // "schema" | "wasm"
	FetchConfig  *FetchConfig   `json:"fetch_config,omitempty"`
	ContentHash  string         `json:"content_hash"`
	IsAIGenerated bool          `json:"is_ai_generated"`
	IsOnHome     bool           `json:"is_on_home"`
	Version      int            `json:"version"`
	CreatedAt    int64          `json:"created_at"`
	UpdatedAt    int64          `json:"updated_at"`
}

// GeneratedComponent corresponds to the frontend GeneratedComponent type.
type GeneratedComponent struct {
	ID       string                    `json:"id"`
	Type     string                    `json:"type"`
	Props    map[string]any            `json:"props"`
	Children []GeneratedComponent      `json:"children,omitempty"`
}

// FetchConfig describes how the tool fetches data.
type FetchConfig struct {
	Source   string            `json:"source"`    // "api" | "channel" | "static"
	URL      string            `json:"url"`
	Channel  string            `json:"channel"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Interval int               `json:"interval"` // auto-refresh interval (seconds)
}

// ExecutionResult is the result of tool execution.
type ExecutionResult struct {
	Schema  *GeneratedComponent `json:"schema"`
	Data    map[string]any      `json:"data"`
	Expires int64               `json:"expires,omitempty"`
}

// GenerateRequest is a tool generation request.
type GenerateRequest struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context"` // "chat" | "space"
}

// GenerateResult is a tool generation response.
type GenerateResult struct {
	ToolID      string             `json:"tool_id"`
	Name        string             `json:"name"`
	IsNew       bool               `json:"is_new"`
	ChatSchema  GeneratedComponent  `json:"chat_schema"`
	SpaceSchema GeneratedComponent  `json:"space_schema"`
}

// ToolEngine is the tool execution engine interface.
type ToolEngine interface {
	Execute(ctx context.Context, tool *DynamicTool, params map[string]any) (*ExecutionResult, error)
	Validate(tool *DynamicTool) error
}
