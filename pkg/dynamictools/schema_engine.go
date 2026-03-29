package dynamictools

import (
	"context"
	"fmt"
	"time"
)

// SchemaEngine is the core execution engine for Phase 1 (schema-driven).
// It takes a tool definition, fetches real data if needed, injects the data
// into the UI schema, and returns the complete schema ready for rendering.
type SchemaEngine struct {
	hostFn *HostFunctions
}

// NewSchemaEngine creates a new SchemaEngine with the given host functions.
func NewSchemaEngine(hostFn *HostFunctions) *SchemaEngine {
	return &SchemaEngine{hostFn: hostFn}
}

// Execute runs the tool in chat mode using tool.ChatSchema.
func (e *SchemaEngine) Execute(ctx context.Context, tool *DynamicTool, params map[string]any) (*ExecutionResult, error) {
	var data map[string]any
	if tool.FetchConfig != nil && tool.FetchConfig.Source != "static" {
		fetched, err := e.fetchData(ctx, tool.FetchConfig, params)
		if err != nil {
			return nil, fmt.Errorf("fetch data: %w", err)
		}
		data = fetched
	} else {
		data = params
	}

	schemaCopy := deepCopyComponent(&tool.ChatSchema)
	injectData(schemaCopy, data)

	var expires int64
	if tool.FetchConfig != nil && tool.FetchConfig.Interval > 0 {
		expires = time.Now().Unix() + int64(tool.FetchConfig.Interval)
	}

	return &ExecutionResult{
		Schema:  schemaCopy,
		Data:    data,
		Expires: expires,
	}, nil
}

// ExecuteSpace runs the tool in space mode using tool.SpaceSchema.
func (e *SchemaEngine) ExecuteSpace(ctx context.Context, tool *DynamicTool, params map[string]any) (*ExecutionResult, error) {
	var data map[string]any
	if tool.FetchConfig != nil && tool.FetchConfig.Source != "static" {
		fetched, err := e.fetchData(ctx, tool.FetchConfig, params)
		if err != nil {
			return nil, fmt.Errorf("fetch data: %w", err)
		}
		data = fetched
	} else {
		data = params
	}

	schemaCopy := deepCopyComponent(&tool.SpaceSchema)
	injectData(schemaCopy, data)

	var expires int64
	if tool.FetchConfig != nil && tool.FetchConfig.Interval > 0 {
		expires = time.Now().Unix() + int64(tool.FetchConfig.Interval)
	}

	return &ExecutionResult{
		Schema:  schemaCopy,
		Data:    data,
		Expires: expires,
	}, nil
}

// Validate checks that chat_schema.type and space_schema.type are non-empty.
func (e *SchemaEngine) Validate(tool *DynamicTool) error {
	if tool.ChatSchema.Type == "" {
		return fmt.Errorf("chat_schema.type is required")
	}
	if tool.SpaceSchema.Type == "" {
		return fmt.Errorf("space_schema.type is required")
	}
	return nil
}

// fetchData retrieves data based on the FetchConfig source.
func (e *SchemaEngine) fetchData(ctx context.Context, fc *FetchConfig, params map[string]any) (map[string]any, error) {
	switch fc.Source {
	case "api":
		method := fc.Method
		if method == "" {
			method = "GET"
		}
		url := fc.URL
		for k, v := range params {
			url = replaceParam(url, k, fmt.Sprintf("%v", v))
		}
		return e.hostFn.HTTPFetch(ctx, method, url, fc.Headers, fc.Body)

	case "channel":
		messages, err := e.hostFn.ReadChannel(ctx, fc.Channel, 10)
		if err != nil {
			return nil, err
		}
		return map[string]any{"messages": messages}, nil

	case "static":
		return params, nil

	default:
		return nil, fmt.Errorf("unknown fetch source: %s", fc.Source)
	}
}

// deepCopyComponent creates a deep clone of a GeneratedComponent tree.
func deepCopyComponent(c *GeneratedComponent) *GeneratedComponent {
	cp := &GeneratedComponent{
		ID:   c.ID,
		Type: c.Type,
		Props: make(map[string]any, len(c.Props)),
	}
	for k, v := range c.Props {
		cp.Props[k] = v
	}
	if len(c.Children) > 0 {
		cp.Children = make([]GeneratedComponent, len(c.Children))
		for i, child := range c.Children {
			cp.Children[i] = *deepCopyComponent(&child)
		}
	}
	return cp
}

// injectData recursively merges data map keys into component props (only
// keys that don't already exist in props).
func injectData(component *GeneratedComponent, data map[string]any) {
	for k, v := range data {
		if _, exists := component.Props[k]; !exists {
			component.Props[k] = v
		}
	}
	for i := range component.Children {
		injectData(&component.Children[i], data)
	}
}

// replaceParam replaces the first occurrence of {{key}} in s with value.
func replaceParam(s, key, value string) string {
	placeholder := "{{" + key + "}}"
	result := s
	for i := 0; i < len(result)-len(placeholder); i++ {
		if result[i:i+len(placeholder)] == placeholder {
			result = result[:i] + value + result[i+len(placeholder):]
			break
		}
	}
	return result
}
