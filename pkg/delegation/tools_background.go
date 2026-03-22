package delegation

import (
	"context"
	"fmt"

	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// DelegateBackgroundTool delegates a task to run in the background
type DelegateBackgroundTool struct {
	ds *DelegationSystem
}

// NewDelegateBackgroundTool creates a new delegate_background tool
func NewDelegateBackgroundTool(ds *DelegationSystem) *DelegateBackgroundTool {
	return &DelegateBackgroundTool{ds: ds}
}

func (t *DelegateBackgroundTool) Name() string {
	return "delegate_background"
}

func (t *DelegateBackgroundTool) Description() string {
	return "Delegate a task to run in the background. The sub-agent will work independently and the result will be delivered when you check back. Use this for long-running tasks that don't need immediate results."
}

func (t *DelegateBackgroundTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "The task to delegate to the background sub-agent",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Optional short label for the sub-agent",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        []string{"research", "code", "analysis", "writing", "general"},
				"description": "Task category (default: auto-detect)",
			},
			"role_prompt": map[string]any{
				"type":        "string",
				"description": "Optional custom role prompt",
			},
			"tools": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional list of tool names for the sub-agent",
			},
		},
		"required": []string{"task"},
	}
}

func (t *DelegateBackgroundTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	task, ok := args["task"].(string)
	if !ok {
		return toolErrorResult("task is required")
	}

	label, _ := args["label"].(string)
	categoryStr, _ := args["category"].(string)
	rolePrompt, _ := args["role_prompt"].(string)
	customTools := toStringSlice(args["tools"])

	userID := tools.DelegationUserID(ctx)
	originChannel := tools.ToolChannel(ctx)
	originChatID := tools.ToolChatID(ctx)

	// Determine category
	category := ClassifyTask(task)
	if categoryStr != "" {
		category = TaskCategory(categoryStr)
	}

	// Find or create template
	template, _, err := t.ds.templates.FindOrCreate(ctx, userID, task, rolePrompt, customTools)
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to find/create template: %v", err))
	}

	// Create sub-agent
	agent, err := t.ds.lifecycle.Create(ctx, userID, label, template.RolePrompt, template.TaskKeywords, template.Tools)
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to create sub-agent: %v", err))
	}

	// Submit background task
	record, err := t.ds.background.Submit(ctx, agent, task, category, template.RolePrompt, template.Tools, originChannel, originChatID)
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to submit background task: %v", err))
	}

	agentLabel := label
	if agentLabel == "" {
		agentLabel = agent.Label
	}

	return toolSuccessResult(
		fmt.Sprintf("Background task submitted successfully.\nTask ID: %s\nSub-agent: %s\nCategory: %s\n\nThe task is running in the background. Results will be available when you check back.", record.ID, agentLabel, category),
		fmt.Sprintf("⏳ Background task submitted to sub-agent '%s'. Check back later for results.", agentLabel),
	)
}

// DelegateToExistingTool sends a task to an existing sub-agent
type DelegateToExistingTool struct {
	ds *DelegationSystem
}

// NewDelegateToExistingTool creates a new delegate_to_existing tool
func NewDelegateToExistingTool(ds *DelegationSystem) *DelegateToExistingTool {
	return &DelegateToExistingTool{ds: ds}
}

func (t *DelegateToExistingTool) Name() string {
	return "delegate_to_existing"
}

func (t *DelegateToExistingTool) Description() string {
	return "Send a task to an existing sub-agent by ID or label. The sub-agent must be active. Use list_sub_agents to see available sub-agents."
}

func (t *DelegateToExistingTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sub_agent_id": map[string]any{
				"type":        "string",
				"description": "The ID or label of the existing sub-agent",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "The task to send to the sub-agent",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "Whether to run in background (default: false)",
			},
		},
		"required": []string{"sub_agent_id", "task"},
	}
}

func (t *DelegateToExistingTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	subAgentID, ok := args["sub_agent_id"].(string)
	if !ok {
		return toolErrorResult("sub_agent_id is required")
	}

	task, ok := args["task"].(string)
	if !ok {
		return toolErrorResult("task is required")
	}

	background := getOptionalBool(args["background"], false)
	delegationUser := tools.DelegationUserID(ctx)
	originChannel := tools.ToolChannel(ctx)
	originChatID := tools.ToolChatID(ctx)

	// Find the agent
	agent, err := t.ds.lifecycle.Get(ctx, subAgentID)
	if err != nil || agent == nil {
		return toolErrorResult(fmt.Sprintf("Sub-agent '%s' not found", subAgentID))
	}

	if agent.Status != StatusActive {
		return toolErrorResult(fmt.Sprintf("Sub-agent '%s' is not active (status: %s)", subAgentID, agent.Status))
	}

	category := ClassifyTask(task)

	if background {
		// Submit as background task
		record, err := t.ds.background.Submit(ctx, agent, task, category, agent.RolePrompt, agent.Tools, originChannel, originChatID)
		if err != nil {
			return toolErrorResult(fmt.Sprintf("Failed to submit background task: %v", err))
		}

		return toolSuccessResult(
			fmt.Sprintf("Background task submitted to existing sub-agent '%s'.\nTask ID: %s", agent.Label, record.ID),
			fmt.Sprintf("⏳ Task submitted to sub-agent '%s'. Running in background.", agent.Label),
		)
	}

	return t.ds.runSerialized(ctx, delegationUser, func(c context.Context) *tools.ToolResult {
		return t.executeExistingSync(c, agent, task, category)
	})
}

func (t *DelegateToExistingTool) executeExistingSync(ctx context.Context, agent *SubAgentRecord, task string, category TaskCategory) *tools.ToolResult {
	userID := tools.DelegationUserID(ctx)
	timeout := t.ds.timeoutEstimator.Estimate(ctx, userID, task, category)
	execCtx, cancel := context.WithTimeout(ctx, timeout.Duration)
	defer cancel()

	executor := NewLLMExecutor(t.ds.store, t.ds.provider, t.ds.model, t.ds.workspace, t.ds.config.DefaultSubAgentTools)
	result, err := executor.Execute(execCtx, agent.SessionKey, agent.RolePrompt, task, agent.Tools)

	if err != nil {
		t.ds.lifecycle.RecordTaskResult(ctx, agent.ID, false, 0)
		return toolErrorResult(fmt.Sprintf("Task execution failed: %v", err))
	}

	t.ds.lifecycle.RecordTaskResult(ctx, agent.ID, true, timeout.Duration.Milliseconds())

	return toolSuccessResult(
		fmt.Sprintf("Sub-agent '%s' completed task.\n\nResult:\n%s", agent.Label, result),
		fmt.Sprintf("✅ Sub-agent '%s' completed the task.", agent.Label),
	)
}
