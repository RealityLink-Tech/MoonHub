package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/adaptive_memory"
)

// MemoryToolProvider interface for memory tool dependency injection
type MemoryToolProvider interface {
	GetAdaptiveMemory() *adaptive_memory.Engine
}

// MemoryTool allows the agent to record and search memories
type MemoryTool struct {
	provider MemoryToolProvider
}

// NewMemoryTool creates a new memory tool
func NewMemoryTool(provider MemoryToolProvider) *MemoryTool {
	return &MemoryTool{
		provider: provider,
	}
}

func (t *MemoryTool) Name() string {
	return "memory"
}

func (t *MemoryTool) Description() string {
	return `Manage agent's adaptive memory system. Use this tool to:
- Record important information about the user (preferences, facts, corrections)
- Search for relevant memories based on queries
- List recent memories

Memory Types:
- correction: User corrections (highest priority, use when user corrects you)
- preference_learned: User preferences learned from conversation
- fact_stored: General facts about the user or context
- task_completed: Records of completed tasks
- user_feedback: Explicit user feedback
- insight: Agent-generated insights
- error_learned: Error patterns to remember`
}

func (t *MemoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"record", "search", "list", "consolidate"},
				"description": "Action to perform: record, search, list, or consolidate",
			},
			"event_type": map[string]any{
				"type":        "string",
				"enum":        []string{"correction", "preference_learned", "fact_stored", "task_completed", "user_feedback", "insight", "error_learned", "reminder", "context_update", "delegation_result"},
				"description": "Type of memory event (required for 'record' action)",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Memory content (required for 'record' action)",
			},
			"outcome": map[string]any{
				"type":        "string",
				"description": "Optional outcome or result (for 'record' action)",
			},
			"importance": map[string]any{
				"type":        "number",
				"description": "Optional importance score 0.0-1.0 (for 'record' action, defaults based on event_type)",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query (required for 'search' action)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results (for 'search' and 'list' actions, default: 10)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *MemoryTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("action is required")
	}

	// Get adaptive memory engine
	engine := t.provider.GetAdaptiveMemory()
	if engine == nil {
		return ErrorResult("Adaptive memory is not available")
	}

	// Get user ID from context
	userID := ToolChatID(ctx)
	if userID == "" {
		userID = "default"
	}

	switch action {
	case "record":
		return t.handleRecord(ctx, engine, userID, args)
	case "search":
		return t.handleSearch(ctx, engine, userID, args)
	case "list":
		return t.handleList(ctx, engine, userID, args)
	case "consolidate":
		return t.handleConsolidate(ctx, engine, userID)
	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s", action))
	}
}

func (t *MemoryTool) handleRecord(ctx context.Context, engine *adaptive_memory.Engine, userID string, args map[string]any) *ToolResult {
	eventTypeStr, _ := args["event_type"].(string)
	if eventTypeStr == "" {
		return ErrorResult("event_type is required for record action")
	}

	content, _ := args["content"].(string)
	if content == "" {
		return ErrorResult("content is required for record action")
	}

	// Create event input
	event := adaptive_memory.EventInput{
		Type:    adaptive_memory.EpisodicEventType(eventTypeStr),
		Content: content,
	}

	// Optional outcome
	if outcome, ok := args["outcome"].(string); ok && outcome != "" {
		event.Outcome = &outcome
	}

	// Optional importance
	if importance, ok := args["importance"].(float64); ok && importance >= 0 && importance <= 1 {
		event.Importance = &importance
	}

	// Record the event
	id, err := engine.RecordEvent(userID, event)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to record memory: %v", err)).WithError(err)
	}

	return SilentResult(fmt.Sprintf("Memory recorded successfully (ID: %s, Type: %s)", id, eventTypeStr))
}

func (t *MemoryTool) handleSearch(ctx context.Context, engine *adaptive_memory.Engine, userID string, args map[string]any) *ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query is required for search action")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	results, err := engine.Search(userID, query, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to search memory: %v", err)).WithError(err)
	}

	if len(results) == 0 {
		return NewToolResult("No memories found matching the query.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d memories:\n\n", len(results)))
	for i, result := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (relevance: %.2f)\n", i+1, result.Source, result.Content, result.RelevanceScore))
	}

	return NewToolResult(sb.String())
}

func (t *MemoryTool) handleList(ctx context.Context, engine *adaptive_memory.Engine, userID string, args map[string]any) *ToolResult {
	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	events, err := engine.GetEvents(userID, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list memories: %v", err)).WithError(err)
	}

	if len(events) == 0 {
		return NewToolResult("No memories recorded yet.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Recent %d memories:\n\n", len(events)))
	for i, event := range events {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (importance: %.2f)\n", i+1, event.EventType, event.Content, event.Importance))
	}

	return NewToolResult(sb.String())
}

func (t *MemoryTool) handleConsolidate(ctx context.Context, engine *adaptive_memory.Engine, userID string) *ToolResult {
	result, err := engine.Consolidate(userID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to consolidate memory: %v", err)).WithError(err)
	}

	return SilentResult(fmt.Sprintf("Memory consolidated: decayed=%d, pruned=%d, merged=%d", result.Decayed, result.Pruned, result.Merged))
}
