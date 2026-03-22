package delegation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// ListSubAgentsTool lists all sub-agents for the user
type ListSubAgentsTool struct {
	ds *DelegationSystem
}

// NewListSubAgentsTool creates a new list_sub_agents tool
func NewListSubAgentsTool(ds *DelegationSystem) *ListSubAgentsTool {
	return &ListSubAgentsTool{ds: ds}
}

func (t *ListSubAgentsTool) Name() string {
	return "list_sub_agents"
}

func (t *ListSubAgentsTool) Description() string {
	return "List all active sub-agents. Shows their status, completed tasks, and success rates."
}

func (t *ListSubAgentsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"include_suspended": map[string]any{
				"type":        "boolean",
				"description": "Include suspended sub-agents in the list (default: false)",
			},
		},
	}
}

func (t *ListSubAgentsTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	userID := tools.DelegationUserID(ctx)
	_ = getOptionalBool(args["include_suspended"], false) // includeSuspended not yet implemented in lifecycle

	agents, err := t.ds.lifecycle.ListActive(ctx, userID)
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to list sub-agents: %v", err))
	}

	if len(agents) == 0 {
		return toolSuccessResult(
			"No active sub-agents.",
			"No active sub-agents.",
		)
	}

	var summaries []string
	var userSummaries []string

	for _, agent := range agents {
		statusEmoji := "🟢"
		if agent.Status == StatusSuspended {
			statusEmoji = "🟡"
		}

		summary := fmt.Sprintf("%s **%s** (%s)\n   ID: %s\n   Tasks: %d | Success Rate: %.0f%%\n   Last Active: %s",
			statusEmoji,
			agent.Label,
			agent.Status,
			agent.ID,
			agent.CompletedTasks,
			agent.SuccessRate*100,
			formatTimeAgo(agent.LastActiveAt),
		)

		summaries = append(summaries, summary)
		userSummaries = append(userSummaries, fmt.Sprintf("%s %s", statusEmoji, agent.Label))
	}

	return toolSuccessResult(
		fmt.Sprintf("Active Sub-Agents (%d):\n\n%s", len(agents), strings.Join(summaries, "\n\n")),
		fmt.Sprintf("Sub-agents: %s", strings.Join(userSummaries, ", ")),
	)
}

// ManageSubAgentTool manages sub-agent lifecycle
type ManageSubAgentTool struct {
	ds *DelegationSystem
}

// NewManageSubAgentTool creates a new manage_sub_agent tool
func NewManageSubAgentTool(ds *DelegationSystem) *ManageSubAgentTool {
	return &ManageSubAgentTool{ds: ds}
}

func (t *ManageSubAgentTool) Name() string {
	return "manage_sub_agent"
}

func (t *ManageSubAgentTool) Description() string {
	return "Manage sub-agent lifecycle: dismiss (soft delete), revive (reactivate), or kill (permanent delete)."
}

func (t *ManageSubAgentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sub_agent_id": map[string]any{
				"type":        "string",
				"description": "The ID or label of the sub-agent",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"dismiss", "revive", "kill"},
				"description": "Action to perform: dismiss (suspend), revive (reactivate), or kill (delete)",
			},
		},
		"required": []string{"sub_agent_id", "action"},
	}
}

func (t *ManageSubAgentTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	subAgentID, ok := args["sub_agent_id"].(string)
	if !ok {
		return toolErrorResult("sub_agent_id is required")
	}

	action, ok := args["action"].(string)
	if !ok {
		return toolErrorResult("action is required")
	}

	var err error
	var message string

	switch action {
	case "dismiss":
		err = t.ds.lifecycle.Dismiss(ctx, subAgentID)
		message = fmt.Sprintf("Sub-agent '%s' dismissed.", subAgentID)

	case "revive":
		err = t.ds.lifecycle.Revive(ctx, subAgentID)
		message = fmt.Sprintf("Sub-agent '%s' revived.", subAgentID)

	case "kill":
		err = t.ds.lifecycle.Kill(ctx, subAgentID)
		message = fmt.Sprintf("Sub-agent '%s' permanently deleted.", subAgentID)

	default:
		return toolErrorResult(fmt.Sprintf("Unknown action: %s. Use 'dismiss', 'revive', or 'kill'.", action))
	}

	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to %s sub-agent: %v", action, err))
	}

	return toolSuccessResult(message, message)
}

// ManageTemplateTool manages role templates
type ManageTemplateTool struct {
	ds *DelegationSystem
}

// NewManageTemplateTool creates a new manage_template tool
func NewManageTemplateTool(ds *DelegationSystem) *ManageTemplateTool {
	return &ManageTemplateTool{ds: ds}
}

func (t *ManageTemplateTool) Name() string {
	return "manage_template"
}

func (t *ManageTemplateTool) Description() string {
	return "Manage role templates: list, delete, or view statistics."
}

func (t *ManageTemplateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "delete", "stats"},
				"description": "Action: list (show templates), delete (remove template), stats (show usage)",
			},
			"template_id": map[string]any{
				"type":        "string",
				"description": "Template ID (required for delete)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *ManageTemplateTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return toolErrorResult("action is required")
	}

	userID := tools.DelegationUserID(ctx)

	switch action {
	case "list":
		templates, err := t.ds.templates.ListTemplates(ctx, userID)
		if err != nil {
			return toolErrorResult(fmt.Sprintf("Failed to list templates: %v", err))
		}

		if len(templates) == 0 {
			return toolSuccessResult("No templates found.", "No templates found.")
		}

		var lines []string
		for _, t := range templates {
			lines = append(lines, fmt.Sprintf("- **%s** (%s)\n  ID: %s | Uses: %d | Success: %d",
				t.Name, t.Category, t.ID, t.SuccessCount+t.FailureCount, t.SuccessCount))
		}

		return toolSuccessResult(
			fmt.Sprintf("Templates (%d):\n\n%s", len(templates), strings.Join(lines, "\n\n")),
			fmt.Sprintf("Found %d templates.", len(templates)),
		)

	case "delete":
		templateID, ok := args["template_id"].(string)
		if !ok {
			return toolErrorResult("template_id is required for delete action")
		}

		if err := t.ds.templates.DeleteTemplate(ctx, templateID); err != nil {
			return toolErrorResult(fmt.Sprintf("Failed to delete template: %v", err))
		}

		return toolSuccessResult(
			fmt.Sprintf("Template '%s' deleted.", templateID),
			fmt.Sprintf("Template deleted."),
		)

	case "stats":
		stats, err := t.ds.templates.GetTemplateStats(ctx, userID)
		if err != nil {
			return toolErrorResult(fmt.Sprintf("Failed to get stats: %v", err))
		}

		return toolSuccessResult(
			fmt.Sprintf("Template Statistics:\n\nTotal Templates: %v\nBy Category: %v\nTop Templates: %v",
				stats["total_templates"], stats["by_category"], stats["top_templates"]),
			fmt.Sprintf("Templates: %v total", stats["total_templates"]),
		)

	default:
		return toolErrorResult(fmt.Sprintf("Unknown action: %s", action))
	}
}

// ConfirmTaskTool confirms and retrieves background task results
type ConfirmTaskTool struct {
	ds *DelegationSystem
}

// NewConfirmTaskTool creates a new confirm_task tool
func NewConfirmTaskTool(ds *DelegationSystem) *ConfirmTaskTool {
	return &ConfirmTaskTool{ds: ds}
}

func (t *ConfirmTaskTool) Name() string {
	return "confirm_task"
}

func (t *ConfirmTaskTool) Description() string {
	return "Check and confirm background task results. Lists undelivered results and marks them as delivered."
}

func (t *ConfirmTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "Specific task ID to confirm (optional, if omitted shows all pending)",
			},
			"mark_delivered": map[string]any{
				"type":        "boolean",
				"description": "Mark the task(s) as delivered (default: true)",
			},
		},
	}
}

func (t *ConfirmTaskTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	userID := tools.DelegationUserID(ctx)
	taskID, _ := args["task_id"].(string)
	markDelivered := getOptionalBool(args["mark_delivered"], true)

	if taskID != "" {
		// Get specific task
		// Note: We'd need to add a GetBackgroundTask method to the store
		// For now, just list all and filter
	}

	// Get all undelivered tasks
	tasks, err := t.ds.GetUndeliveredResults(ctx, userID)
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to get undelivered tasks: %v", err))
	}

	if len(tasks) == 0 {
		return toolSuccessResult(
			"No pending background task results.",
			"No pending results.",
		)
	}

	var results []string
	for _, task := range tasks {
		if taskID != "" && task.ID != taskID {
			continue
		}

		results = append(results, FormatTaskResultForLLM(task))

		if markDelivered {
			t.ds.MarkDelivered(ctx, task.ID)
		}
	}

	if len(results) == 0 {
		return toolSuccessResult(
			"No matching task found.",
			"No matching task.",
		)
	}

	return toolSuccessResult(
		fmt.Sprintf("Background Task Results:\n\n%s", strings.Join(results, "\n\n---\n\n")),
		fmt.Sprintf("Received %d background task result(s).", len(results)),
	)
}

// Helper function to format time ago
func formatTimeAgo(timestamp int64) string {
	if timestamp == 0 {
		return "never"
	}

	now := timeNow()
	duration := now - timestamp

	seconds := duration / 1000
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%d days ago", days)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours ago", hours)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	return "just now"
}

// timeNow is a variable for testing
var timeNow = func() int64 {
	return time.Now().UnixMilli()
}
