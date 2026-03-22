package delegation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// DelegationSystem is the main facade for the delegation system
type DelegationSystem struct {
	store            DelegationStore
	config           DelegationConfig
	lifecycle        *LifecycleManager
	templates        *TemplateManager
	background       *BackgroundRunner
	timeoutEstimator *TimeoutEstimator
	intercom         *Intercom
	blackboard       *Blackboard
	queue            *SessionQueue

	// For sub-agent execution
	provider  providers.LLMProvider
	model     string
	workspace string
}

// NewDelegationSystem creates a new delegation system
func NewDelegationSystem(dbPath string, config DelegationConfig, provider providers.LLMProvider, model, workspace string) (*DelegationSystem, error) {
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegation store: %w", err)
	}

	intercom := NewIntercom(100)
	blackboard := NewBlackboard(50)
	blackboard.SetIntercom(intercom)
	timeoutEstimator := NewTimeoutEstimator(store)
	lifecycle := NewLifecycleManager(store, config, intercom)
	templates := NewTemplateManager(store, config.DefaultSubAgentTools)
	background := NewBackgroundRunner(store, lifecycle, timeoutEstimator, intercom, config)
	queue := NewSessionQueue()

	// Set up executor
	executor := NewLLMExecutor(store, provider, model, workspace, config.DefaultSubAgentTools)
	background.SetExecutor(executor)

	return &DelegationSystem{
		store:            store,
		config:           config,
		lifecycle:        lifecycle,
		templates:        templates,
		background:       background,
		timeoutEstimator: timeoutEstimator,
		intercom:         intercom,
		blackboard:       blackboard,
		queue:            queue,
		provider:         provider,
		model:            model,
		workspace:        workspace,
	}, nil
}

// RegisterTools registers all delegation tools to the tool registry
func (ds *DelegationSystem) RegisterTools(registry *tools.ToolRegistry) {
	registry.Register(NewDelegateTaskTool(ds))
	registry.Register(NewDelegateTasksTool(ds))
	registry.Register(NewDelegateBackgroundTool(ds))
	registry.Register(NewDelegateToExistingTool(ds))
	registry.Register(NewListSubAgentsTool(ds))
	registry.Register(NewManageSubAgentTool(ds))
	registry.Register(NewManageTemplateTool(ds))
	registry.Register(NewConfirmTaskTool(ds))
}

// Close closes the delegation system
func (ds *DelegationSystem) Close() error {
	return ds.store.Close()
}

// GetUndeliveredResults gets undelivered background task results
func (ds *DelegationSystem) GetUndeliveredResults(ctx context.Context, userID string) ([]*BackgroundTaskRecord, error) {
	return ds.background.GetUndeliveredResults(ctx, userID)
}

// MarkDelivered marks a task as delivered
func (ds *DelegationSystem) MarkDelivered(ctx context.Context, taskID string) error {
	return ds.background.MarkDelivered(ctx, taskID)
}

// OnTaskCompleted subscribes to task completion events
func (ds *DelegationSystem) OnTaskCompleted(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicTaskCompleted, handler)
}

// OnTaskFailed subscribes to task failure events
func (ds *DelegationSystem) OnTaskFailed(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicTaskFailed, handler)
}

// OnTaskQueued subscribes to task queued events
func (ds *DelegationSystem) OnTaskQueued(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicTaskQueued, handler)
}

// OnAgentCreated subscribes to agent creation events
func (ds *DelegationSystem) OnAgentCreated(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicAgentCreated, handler)
}

// OnAgentDismissed subscribes to agent dismissal events
func (ds *DelegationSystem) OnAgentDismissed(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicAgentDismissed, handler)
}

// OnAgentSuspended subscribes to agent suspension events
func (ds *DelegationSystem) OnAgentSuspended(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicAgentSuspended, handler)
}

// OnAgentRevived subscribes to agent revival events
func (ds *DelegationSystem) OnAgentRevived(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicAgentRevived, handler)
}

// OnMemoryUpdated subscribes to memory update events
func (ds *DelegationSystem) OnMemoryUpdated(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicMemoryUpdated, handler)
}

// OnMemoryConsolidated subscribes to memory consolidation events
func (ds *DelegationSystem) OnMemoryConsolidated(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicMemoryConsolidated, handler)
}

// OnBlackboardProposal subscribes to blackboard proposal events
func (ds *DelegationSystem) OnBlackboardProposal(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicBlackboardProposal, handler)
}

// OnBlackboardResolved subscribes to blackboard resolution events
func (ds *DelegationSystem) OnBlackboardResolved(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicBlackboardResolved, handler)
}

// OnNudgeScheduled subscribes to nudge scheduled events
func (ds *DelegationSystem) OnNudgeScheduled(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicNudgeScheduled, handler)
}

// OnNudgeDelivered subscribes to nudge delivered events
func (ds *DelegationSystem) OnNudgeDelivered(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicNudgeDelivered, handler)
}

// OnNudgeSuppressed subscribes to nudge suppressed events
func (ds *DelegationSystem) OnNudgeSuppressed(handler EventHandler) UnsubscribeFunc {
	return ds.intercom.On(TopicNudgeSuppressed, handler)
}

// BlackboardPost records a proposal and emits blackboard:proposal when intercom is wired.
func (ds *DelegationSystem) BlackboardPost(userID, subAgentID, proposal string) {
	ds.blackboard.Post(userID, subAgentID, proposal)
}

// BlackboardResolve resolves a proposal and emits blackboard:resolved.
func (ds *DelegationSystem) BlackboardResolve(userID, subAgentID, resolution string) {
	ds.blackboard.Resolve(userID, subAgentID, resolution)
}

// EmitTaskQueued emits a task queued event
func (ds *DelegationSystem) EmitTaskQueued(ctx context.Context, userID, taskID string, data map[string]any) {
	ds.intercom.Emit(ctx, TopicTaskQueued, userID, mergeMap(data, map[string]any{"task_id": taskID}))
}

// EmitMemoryUpdated emits a memory updated event
func (ds *DelegationSystem) EmitMemoryUpdated(ctx context.Context, userID string, data map[string]any) {
	ds.intercom.Emit(ctx, TopicMemoryUpdated, userID, data)
}

// EmitMemoryConsolidated emits a memory consolidated event
func (ds *DelegationSystem) EmitMemoryConsolidated(ctx context.Context, userID string, data map[string]any) {
	ds.intercom.Emit(ctx, TopicMemoryConsolidated, userID, data)
}

// EmitNudgeScheduled emits a nudge scheduled event
func (ds *DelegationSystem) EmitNudgeScheduled(ctx context.Context, userID string, data map[string]any) {
	ds.intercom.Emit(ctx, TopicNudgeScheduled, userID, data)
}

// EmitNudgeDelivered emits a nudge delivered event
func (ds *DelegationSystem) EmitNudgeDelivered(ctx context.Context, userID string, data map[string]any) {
	ds.intercom.Emit(ctx, TopicNudgeDelivered, userID, data)
}

// EmitNudgeSuppressed emits a nudge suppressed event
func (ds *DelegationSystem) EmitNudgeSuppressed(ctx context.Context, userID string, data map[string]any) {
	ds.intercom.Emit(ctx, TopicNudgeSuppressed, userID, data)
}

// mergeMap merges two maps (helper function)
func mergeMap(m1, m2 map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}

// runSerialized runs synchronous delegation work through the per-partition session queue.
func (ds *DelegationSystem) runSerialized(ctx context.Context, userID string, work func(context.Context) *tools.ToolResult) *tools.ToolResult {
	key := strings.TrimSpace(userID)
	if key == "" {
		key = "default"
	}
	var out *tools.ToolResult
	done := ds.queue.Submit(ctx, key, generateID("dq"), func(c context.Context) error {
		out = work(c)
		return nil
	})
	if err := <-done; err != nil {
		if errors.Is(err, ErrQueueFull) {
			return toolErrorResult("delegation session queue is full; try again later")
		}
	}
	return out
}

// CleanupOldData removes old data based on retention policy
func (ds *DelegationSystem) CleanupOldData(ctx context.Context) error {
	return ds.lifecycle.CleanupOldData(ctx)
}

// Stats returns statistics about the delegation system
func (ds *DelegationSystem) Stats() map[string]any {
	return map[string]any{
		"background": ds.background.Stats(),
		"queue":      ds.queue.Stats(),
		"intercom":   ds.intercom.Stats(),
		"enabled":    true,
	}
}

// LLMExecutor executes tasks using the LLM
type LLMExecutor struct {
	store        DelegationStore
	provider     providers.LLMProvider
	model        string
	workspace    string
	defaultTools []string
}

// NewLLMExecutor creates a new LLM executor
func NewLLMExecutor(
	store DelegationStore,
	provider providers.LLMProvider,
	model string,
	workspace string,
	defaultTools []string,
) *LLMExecutor {
	return &LLMExecutor{
		store:        store,
		provider:     provider,
		model:        model,
		workspace:    workspace,
		defaultTools: defaultTools,
	}
}

// Execute executes a task with the given role prompt
func (e *LLMExecutor) Execute(ctx context.Context, sessionKey, rolePrompt, task string, toolNames []string) (string, error) {
	// Build system prompt
	systemPrompt := fmt.Sprintf(`%s

You are a specialized sub-agent working on a delegated task.
Complete the task independently and provide a clear summary of your work.

Working directory: %s`, rolePrompt, e.workspace)

	// Build messages
	messages := []providers.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: task,
		},
	}

	// Execute
	response, err := e.provider.Chat(ctx, messages, nil, e.model, map[string]any{
		"max_tokens": 4096,
	})
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// Helper functions

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// GetDBPath returns the default database path for a workspace
func GetDBPath(workspace string) string {
	return filepath.Join(workspace, "delegation.db")
}

// FormatTaskResultForUser formats a task result for user display
func FormatTaskResultForUser(record *BackgroundTaskRecord) string {
	status := "✅"
	if record.Status == "failed" {
		status = "❌"
	}

	return fmt.Sprintf("%s **Background Task %s**\n\n%s",
		status,
		record.Category,
		record.Result,
	)
}

// FormatTaskResultForLLM formats a task result for LLM context injection
func FormatTaskResultForLLM(record *BackgroundTaskRecord) string {
	status := "completed"
	if record.Status == "failed" {
		status = "failed"
	}

	category := record.Category
	if category == "" {
		category = "general"
	}

	return fmt.Sprintf("**Task ID**: %s\n**Category**: %s\n**Status**: %s\n**Result**:\n%s",
		record.ID,
		category,
		status,
		record.Result,
	)
}

// toolErrorResult creates an error tool result
func toolErrorResult(msg string) *tools.ToolResult {
	return &tools.ToolResult{
		ForLLM:  msg,
		ForUser: msg,
		IsError: true,
	}
}

// toolSuccessResult creates a successful tool result
func toolSuccessResult(llmMsg, userMsg string) *tools.ToolResult {
	return &tools.ToolResult{
		ForLLM:  llmMsg,
		ForUser: userMsg,
	}
}
