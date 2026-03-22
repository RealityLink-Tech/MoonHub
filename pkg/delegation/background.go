package delegation

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

// SubAgentExecutor defines the interface for executing sub-agent tasks
type SubAgentExecutor interface {
	Execute(ctx context.Context, sessionKey, rolePrompt, task string, tools []string) (string, error)
}

// BackgroundRunner manages background task execution
type BackgroundRunner struct {
	store            DelegationStore
	lifecycle        *LifecycleManager
	timeoutEstimator *TimeoutEstimator
	intercom         *Intercom
	config           DelegationConfig

	// Execution tracking
	running   atomic.Int64
	taskQueue chan *backgroundTask

	// Executor function
	executor SubAgentExecutor
}

type backgroundTask struct {
	record     *BackgroundTaskRecord
	agent      *SubAgentRecord
	rolePrompt string
	tools      []string
}

// NewBackgroundRunner creates a new background runner
func NewBackgroundRunner(
	store DelegationStore,
	lifecycle *LifecycleManager,
	timeoutEstimator *TimeoutEstimator,
	intercom *Intercom,
	config DelegationConfig,
) *BackgroundRunner {
	runner := &BackgroundRunner{
		store:            store,
		lifecycle:        lifecycle,
		timeoutEstimator: timeoutEstimator,
		intercom:         intercom,
		config:           config,
		taskQueue:        make(chan *backgroundTask, 100),
	}

	// Start worker goroutines
	for i := 0; i < config.MaxConcurrentTasks; i++ {
		go runner.worker()
	}

	return runner
}

// SetExecutor sets the task executor
func (r *BackgroundRunner) SetExecutor(executor SubAgentExecutor) {
	r.executor = executor
}

// Submit submits a background task for execution
func (r *BackgroundRunner) Submit(ctx context.Context, agent *SubAgentRecord, task string, category TaskCategory, rolePrompt string, tools []string, originChannel, originChatID string) (*BackgroundTaskRecord, error) {
	// Check concurrent task limit
	if int(r.running.Load()) >= r.config.MaxConcurrentTasks {
		return nil, fmt.Errorf("max concurrent tasks (%d) reached", r.config.MaxConcurrentTasks)
	}

	now := time.Now().UnixMilli()
	record := &BackgroundTaskRecord{
		ID:            generateID("task"),
		SubAgentID:    agent.ID,
		UserID:        agent.UserID,
		Task:          task,
		Category:      category,
		Status:        "pending",
		CreatedAt:     now,
		OriginChannel: originChannel,
		OriginChatID:  originChatID,
	}

	if err := r.store.SaveBackgroundTask(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to save background task: %w", err)
	}

	// Emit task:queued event
	r.intercom.Emit(ctx, TopicTaskQueued, agent.UserID, map[string]any{
		"task_id":    record.ID,
		"agent_id":   agent.ID,
		"task":       task,
		"category":   category,
		"created_at": now,
	})

	// Queue the task
	bgTask := &backgroundTask{
		record:     record,
		agent:      agent,
		rolePrompt: rolePrompt,
		tools:      tools,
	}

	select {
	case r.taskQueue <- bgTask:
		logger.InfoCF("delegation", "Submitted background task", map[string]any{
			"task_id":  record.ID,
			"agent_id": agent.ID,
		})
		return record, nil
	default:
		// Queue full
		r.store.UpdateBackgroundTask(ctx, record.ID, map[string]any{
			"status": "failed",
			"result": "Task queue full",
		})
		return nil, fmt.Errorf("task queue full")
	}
}

// worker processes tasks from the queue
func (r *BackgroundRunner) worker() {
	for task := range r.taskQueue {
		r.running.Add(1)
		r.executeTask(task)
		r.running.Add(-1)
	}
}

// executeTask executes a single background task
func (r *BackgroundRunner) executeTask(task *backgroundTask) {
	ctx := context.Background()

	// Update status to running
	r.store.UpdateBackgroundTask(ctx, task.record.ID, map[string]any{
		"status": "running",
	})

	// Estimate timeout
	timeout := r.timeoutEstimator.Estimate(ctx, task.record.UserID, task.record.Task, task.record.Category)

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout.Duration)
	defer cancel()

	// Record start time
	startTime := time.Now()

	logger.InfoCF("delegation", "Starting background task", map[string]any{
		"task_id":  task.record.ID,
		"agent_id": task.agent.ID,
		"timeout":  timeout.Duration.String(),
	})

	var result string
	var err error

	// Execute the task
	if r.executor != nil {
		result, err = r.executor.Execute(execCtx, task.agent.SessionKey, task.rolePrompt, task.record.Task, task.tools)
	} else {
		err = fmt.Errorf("no executor configured")
	}

	duration := time.Since(startTime)
	now := time.Now().UnixMilli()

	// Update task record
	if err != nil {
		errMsg := err.Error()
		if execCtx.Err() == context.DeadlineExceeded {
			errMsg = fmt.Sprintf("Task timed out after %s", timeout.Duration)
		}

		r.store.UpdateBackgroundTask(ctx, task.record.ID, map[string]any{
			"status":       "failed",
			"result":       errMsg,
			"completed_at": now,
		})

		// Record metric
		r.timeoutEstimator.RecordMetric(ctx, task.record.UserID, task.record.Task, duration, false)

		// Record task result
		r.lifecycle.RecordTaskResult(ctx, task.agent.ID, false, duration.Milliseconds())

		// Emit event
		r.intercom.Emit(ctx, TopicTaskFailed, task.record.UserID, map[string]any{
			"task_id":      task.record.ID,
			"sub_agent_id": task.agent.ID,
			"error":        errMsg,
			"duration_ms":  duration.Milliseconds(),
		})

		logger.ErrorCF("delegation", "Background task failed", map[string]any{
			"task_id": task.record.ID,
			"error":   errMsg,
		})
	} else {
		r.store.UpdateBackgroundTask(ctx, task.record.ID, map[string]any{
			"status":       "completed",
			"result":       result,
			"completed_at": now,
		})

		// Record metric
		r.timeoutEstimator.RecordMetric(ctx, task.record.UserID, task.record.Task, duration, true)

		// Record task result
		r.lifecycle.RecordTaskResult(ctx, task.agent.ID, true, duration.Milliseconds())

		// Emit event
		r.intercom.Emit(ctx, TopicTaskCompleted, task.record.UserID, map[string]any{
			"task_id":      task.record.ID,
			"sub_agent_id": task.agent.ID,
			"result":       result,
			"duration_ms":  duration.Milliseconds(),
		})

		logger.InfoCF("delegation", "Background task completed", map[string]any{
			"task_id":     task.record.ID,
			"duration_ms": duration.Milliseconds(),
		})
	}
}

// GetUndeliveredResults gets completed tasks that haven't been delivered
func (r *BackgroundRunner) GetUndeliveredResults(ctx context.Context, userID string) ([]*BackgroundTaskRecord, error) {
	return r.store.GetUndeliveredTasks(ctx, userID)
}

// MarkDelivered marks a task result as delivered
func (r *BackgroundRunner) MarkDelivered(ctx context.Context, taskID string) error {
	return r.store.MarkTaskDelivered(ctx, taskID)
}

// GetRunningCount returns the number of currently running tasks
func (r *BackgroundRunner) GetRunningCount() int {
	return int(r.running.Load())
}

// Stats returns statistics about the background runner
func (r *BackgroundRunner) Stats() map[string]any {
	return map[string]any{
		"running_tasks":  r.running.Load(),
		"queue_size":     len(r.taskQueue),
		"max_concurrent": r.config.MaxConcurrentTasks,
	}
}

// SimpleSubAgentExecutor is a basic implementation of SubAgentExecutor
type SimpleSubAgentExecutor struct {
	provider providers.LLMProvider
	model    string
}

// NewSimpleSubAgentExecutor creates a simple executor using the LLM provider
func NewSimpleSubAgentExecutor(provider providers.LLMProvider, model string) *SimpleSubAgentExecutor {
	return &SimpleSubAgentExecutor{
		provider: provider,
		model:    model,
	}
}

// Execute executes a task using the LLM
func (e *SimpleSubAgentExecutor) Execute(ctx context.Context, sessionKey, rolePrompt, task string, tools []string) (string, error) {
	// Build messages
	messages := []providers.Message{
		{
			Role:    "system",
			Content: rolePrompt,
		},
		{
			Role:    "user",
			Content: task,
		},
	}

	// Execute without tools for now (can be extended)
	response, err := e.provider.Chat(ctx, messages, nil, e.model, map[string]any{
		"max_tokens": 4096,
	})
	if err != nil {
		return "", err
	}

	return response.Content, nil
}
