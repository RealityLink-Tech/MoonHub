package delegation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// DelegateTaskTool delegates a single task to a sub-agent
type DelegateTaskTool struct {
	ds *DelegationSystem
}

// NewDelegateTaskTool creates a new delegate_task tool
func NewDelegateTaskTool(ds *DelegationSystem) *DelegateTaskTool {
	return &DelegateTaskTool{ds: ds}
}

func (t *DelegateTaskTool) Name() string {
	return "delegate_task"
}

func (t *DelegateTaskTool) Description() string {
	return "Delegate a single task to a sub-agent. The sub-agent will work independently and return results."
}

func (t *DelegateTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "The task to delegate to the sub-agent",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Optional short label for the sub-agent",
			},
			"category": map[string]any{
				"type":        "string",
				"enum":        []string{"research", "code", "analysis", "writing", "general"},
				"description": "Task category for timeout estimation (default: auto-detect)",
			},
			"role_prompt": map[string]any{
				"type":        "string",
				"description": "Optional custom role prompt for the sub-agent",
			},
			"tools": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional list of tool names the sub-agent can use (default: safe tools)",
			},
			"reuse_existing": map[string]any{
				"type":        "boolean",
				"description": "Whether to reuse an existing sub-agent if one matches (default: true)",
			},
			"timeout_hint": map[string]any{
				"type":        "integer",
				"description": "Optional timeout in seconds (0 = auto-estimate)",
			},
		},
		"required": []string{"task"},
	}
}

func (t *DelegateTaskTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	uid := tools.DelegationUserID(ctx)
	return t.ds.runSerialized(ctx, uid, func(c context.Context) *tools.ToolResult {
		return t.executeSync(c, args)
	})
}

func (t *DelegateTaskTool) executeSync(ctx context.Context, args map[string]any) *tools.ToolResult {
	task, ok := args["task"].(string)
	if !ok {
		return toolErrorResult("task is required")
	}

	label, _ := args["label"].(string)
	categoryStr, _ := args["category"].(string)
	rolePrompt, _ := args["role_prompt"].(string)
	customTools := toStringSlice(args["tools"])
	reuseExisting := getOptionalBool(args["reuse_existing"], true)
	timeoutHint, _ := args["timeout_hint"].(int)

	userID := tools.DelegationUserID(ctx)

	// Determine category
	category := ClassifyTask(task)
	if categoryStr != "" {
		category = TaskCategory(categoryStr)
	}

	// Extract keywords
	keywords := ExtractKeywords(task)

	// Find or create template
	template, _, err := t.ds.templates.FindOrCreate(ctx, userID, task, rolePrompt, customTools)
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to find/create template: %v", err))
	}

	// Find or create sub-agent
	var agent *SubAgentRecord
	if reuseExisting {
		agent, _ = t.ds.lifecycle.FindReusable(ctx, userID, keywords)
	}

	if agent == nil {
		agent, err = t.ds.lifecycle.Create(ctx, userID, label, template.RolePrompt, template.TaskKeywords, template.Tools)
		if err != nil {
			return toolErrorResult(fmt.Sprintf("Failed to create sub-agent: %v", err))
		}
	}

	// Get timeout estimate
	timeout := t.ds.timeoutEstimator.EstimateWithHint(ctx, userID, task, category, timeoutHint)

	// Execute synchronously
	execCtx, cancel := context.WithTimeout(ctx, timeout.Duration)
	defer cancel()

	executor := NewLLMExecutor(t.ds.store, t.ds.provider, t.ds.model, t.ds.workspace, template.Tools)
	result, err := executor.Execute(execCtx, agent.SessionKey, template.RolePrompt, task, template.Tools)

	if err != nil {
		t.ds.lifecycle.RecordTaskResult(ctx, agent.ID, false, 0)
		return toolErrorResult(fmt.Sprintf("Task execution failed: %v", err))
	}

	// Record success
	t.ds.lifecycle.RecordTaskResult(ctx, agent.ID, true, timeout.Duration.Milliseconds())

	agentLabel := label
	if agentLabel == "" {
		agentLabel = agent.Label
	}

	return toolSuccessResult(
		fmt.Sprintf("Sub-agent '%s' completed task successfully.\n\nResult:\n%s", agentLabel, result),
		fmt.Sprintf("✅ Sub-agent '%s' completed the task.", agentLabel),
	)
}

// DelegateTasksTool delegates multiple tasks in batch
type DelegateTasksTool struct {
	ds *DelegationSystem
}

// NewDelegateTasksTool creates a new delegate_tasks tool
func NewDelegateTasksTool(ds *DelegationSystem) *DelegateTasksTool {
	return &DelegateTasksTool{ds: ds}
}

func (t *DelegateTasksTool) Name() string {
	return "delegate_tasks"
}

func (t *DelegateTasksTool) Description() string {
	return "Delegate multiple tasks (up to 10) to sub-agents in parallel. Each task gets its own sub-agent."
}

func (t *DelegateTasksTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tasks": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task": map[string]any{
							"type":        "string",
							"description": "Task description",
						},
						"label": map[string]any{
							"type":        "string",
							"description": "Optional label for this task",
						},
						"category": map[string]any{
							"type":        "string",
							"enum":        []string{"research", "code", "analysis", "writing", "general"},
							"description": "Task category (default: auto-detect)",
						},
					},
				},
				"description": "List of tasks to delegate",
				"maxItems":    10,
			},
			"parallel": map[string]any{
				"type":        "boolean",
				"description": "Execute tasks in parallel (default: true)",
			},
		},
		"required": []string{"tasks"},
	}
}

type batchTask struct {
	task     string
	label    string
	category TaskCategory
}

type batchResult struct {
	label   string
	result  string
	success bool
}

func (t *DelegateTasksTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	tasksRaw, ok := args["tasks"]
	if !ok {
		return toolErrorResult("tasks is required")
	}

	tasksArr, ok := tasksRaw.([]any)
	if !ok {
		return toolErrorResult("tasks must be an array")
	}

	if len(tasksArr) > 10 {
		return toolErrorResult("Maximum 10 tasks allowed")
	}

	if len(tasksArr) == 0 {
		return toolErrorResult("At least one task is required")
	}

	parallel := getOptionalBool(args["parallel"], true)
	userID := tools.DelegationUserID(ctx)

	// Parse tasks
	var batchTasks []batchTask

	for i, taskItem := range tasksArr {
		taskMap, ok := taskItem.(map[string]any)
		if !ok {
			return toolErrorResult(fmt.Sprintf("Task %d must be an object", i+1))
		}

		taskStr, ok := taskMap["task"].(string)
		if !ok {
			return toolErrorResult(fmt.Sprintf("Task %d is missing 'task' field", i+1))
		}

		label, _ := taskMap["label"].(string)
		categoryStr, _ := taskMap["category"].(string)

		category := ClassifyTask(taskStr)
		if categoryStr != "" {
			category = TaskCategory(categoryStr)
		}

		batchTasks = append(batchTasks, batchTask{
			task:     taskStr,
			label:    label,
			category: category,
		})
	}

	if !parallel {
		return t.ds.runSerialized(ctx, userID, func(c context.Context) *tools.ToolResult {
			results := make([]batchResult, len(batchTasks))
			for i, bt := range batchTasks {
				results[i] = t.executeSingleTask(c, userID, bt)
			}
			return t.formatBatchResults(results)
		})
	}

	results := make([]batchResult, len(batchTasks))
	var wg sync.WaitGroup
	for i, bt := range batchTasks {
		wg.Add(1)
		go func(idx int, bt batchTask) {
			defer wg.Done()
			results[idx] = t.executeSingleTask(ctx, userID, bt)
		}(i, bt)
	}
	wg.Wait()

	return t.formatBatchResults(results)
}

func (t *DelegateTasksTool) formatBatchResults(results []batchResult) *tools.ToolResult {
	var successCount, failCount int
	var summaries []string

	for _, r := range results {
		status := "✅"
		if !r.success {
			status = "❌"
			failCount++
		} else {
			successCount++
		}
		summaries = append(summaries, fmt.Sprintf("%s **%s**: %s", status, r.label, r.result))
	}

	return toolSuccessResult(
		fmt.Sprintf("Batch delegation completed: %d succeeded, %d failed\n\n%s",
			successCount, failCount, summaries),
		fmt.Sprintf("Batch complete: %d succeeded, %d failed", successCount, failCount),
	)
}

func (t *DelegateTasksTool) executeSingleTask(ctx context.Context, userID string, bt batchTask) batchResult {
	resultLabel := bt.label
	if resultLabel == "" {
		resultLabel = fmt.Sprintf("Task-%d", time.Now().UnixMilli()%10000)
	}

	// Find or create template
	template, _, err := t.ds.templates.FindOrCreate(ctx, userID, bt.task, "", nil)
	if err != nil {
		return batchResult{label: resultLabel, result: err.Error(), success: false}
	}

	// Create sub-agent
	agent, err := t.ds.lifecycle.Create(ctx, userID, resultLabel, template.RolePrompt, template.TaskKeywords, template.Tools)
	if err != nil {
		return batchResult{label: resultLabel, result: err.Error(), success: false}
	}

	// Execute
	timeout := t.ds.timeoutEstimator.Estimate(ctx, userID, bt.task, bt.category)
	execCtx, cancel := context.WithTimeout(ctx, timeout.Duration)
	defer cancel()

	executor := NewLLMExecutor(t.ds.store, t.ds.provider, t.ds.model, t.ds.workspace, template.Tools)
	result, err := executor.Execute(execCtx, agent.SessionKey, template.RolePrompt, bt.task, template.Tools)

	if err != nil {
		t.ds.lifecycle.RecordTaskResult(ctx, agent.ID, false, 0)
		return batchResult{label: resultLabel, result: err.Error(), success: false}
	}

	t.ds.lifecycle.RecordTaskResult(ctx, agent.ID, true, timeout.Duration.Milliseconds())

	// Truncate result for batch display
	if len(result) > 200 {
		result = result[:200] + "..."
	}

	return batchResult{label: resultLabel, result: result, success: true}
}
