package delegation

import (
	"context"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// SessionQueue manages per-agent serial execution of tasks
type SessionQueue struct {
	mu     sync.Mutex
	queues map[string]*agentQueue
}

type agentQueue struct {
	tasks   chan *queuedTask
	running bool
}

type queuedTask struct {
	task    func(ctx context.Context) error
	ctx     context.Context
	done    chan error
	taskID  string
	agentID string
}

// NewSessionQueue creates a new session queue
func NewSessionQueue() *SessionQueue {
	return &SessionQueue{
		queues: make(map[string]*agentQueue),
	}
}

// Submit submits a task to an agent's queue
// Tasks for the same agent are executed serially
// Returns a channel that receives the error when complete
func (q *SessionQueue) Submit(ctx context.Context, agentID string, taskID string, task func(ctx context.Context) error) <-chan error {
	q.mu.Lock()

	queue, exists := q.queues[agentID]
	if !exists {
		queue = &agentQueue{
			tasks:   make(chan *queuedTask, 100),
			running: false,
		}
		q.queues[agentID] = queue
	}

	qt := &queuedTask{
		task:    task,
		ctx:     ctx,
		done:    make(chan error, 1),
		taskID:  taskID,
		agentID: agentID,
	}

	// Start processing if not already running
	if !queue.running {
		queue.running = true
		go q.processQueue(agentID)
	}

	q.mu.Unlock()

	// Submit to channel
	select {
	case queue.tasks <- qt:
		logger.DebugCF("delegation", "Task queued", map[string]any{
			"task_id":  taskID,
			"agent_id": agentID,
		})
	default:
		// Queue full, return error
		qt.done <- ErrQueueFull
		close(qt.done)
	}

	return qt.done
}

// processQueue processes tasks for an agent serially
func (q *SessionQueue) processQueue(agentID string) {
	q.mu.Lock()
	queue, exists := q.queues[agentID]
	if !exists {
		return
	}
	q.mu.Unlock()

	for {
		select {
		case task := <-queue.tasks:
			// Execute task
			start := time.Now()
			err := task.task(task.ctx)
			duration := time.Since(start)

			logger.DebugCF("delegation", "Task executed", map[string]any{
				"task_id":     task.taskID,
				"agent_id":    agentID,
				"duration_ms": duration.Milliseconds(),
				"error":       err != nil,
			})

			// Send result
			task.done <- err
			close(task.done)

		default:
			// No more tasks, stop processing
			q.mu.Lock()
			if len(queue.tasks) == 0 {
				queue.running = false
				q.mu.Unlock()
				return
			}
			q.mu.Unlock()
		}
	}
}

// GetQueueLength returns the number of pending tasks for an agent
func (q *SessionQueue) GetQueueLength(agentID string) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	if queue, exists := q.queues[agentID]; exists {
		return len(queue.tasks)
	}
	return 0
}

// GetTotalQueueLength returns total pending tasks across all agents
func (q *SessionQueue) GetTotalQueueLength() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	total := 0
	for _, queue := range q.queues {
		total += len(queue.tasks)
	}
	return total
}

// GetActiveAgents returns the number of agents with active queues
func (q *SessionQueue) GetActiveAgents() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queues)
}

// Clear clears all queues
func (q *SessionQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, queue := range q.queues {
		// Drain remaining tasks with cancellation error
		for {
			select {
			case task := <-queue.tasks:
				task.done <- ErrQueueCleared
				close(task.done)
			default:
				goto done
			}
		}
	done:
		queue.running = false
	}

	q.queues = make(map[string]*agentQueue)
}

// Stats returns statistics about the session queue
func (q *SessionQueue) Stats() map[string]any {
	q.mu.Lock()
	defer q.mu.Unlock()

	agentStats := make(map[string]any)
	for agentID, queue := range q.queues {
		agentStats[agentID] = map[string]any{
			"pending": len(queue.tasks),
			"running": queue.running,
		}
	}

	return map[string]any{
		"active_agents": len(q.queues),
		"agents":        agentStats,
	}
}

// Error definitions
var (
	ErrQueueFull    = &QueueError{Message: "queue is full"}
	ErrQueueCleared = &QueueError{Message: "queue was cleared"}
)

// QueueError represents a queue error
type QueueError struct {
	Message string
}

func (e *QueueError) Error() string {
	return e.Message
}

// PrioritySessionQueue extends SessionQueue with priority support
type PrioritySessionQueue struct {
	mu     sync.Mutex
	high   map[string]*agentQueue
	normal map[string]*agentQueue
}

// NewPrioritySessionQueue creates a new priority session queue
func NewPrioritySessionQueue() *PrioritySessionQueue {
	return &PrioritySessionQueue{
		high:   make(map[string]*agentQueue),
		normal: make(map[string]*agentQueue),
	}
}

// Submit submits a task with normal priority
func (q *PrioritySessionQueue) Submit(ctx context.Context, agentID string, taskID string, task func(ctx context.Context) error) <-chan error {
	return q.SubmitWithPriority(ctx, agentID, taskID, task, false)
}

// SubmitWithPriority submits a task with specified priority
func (q *PrioritySessionQueue) SubmitWithPriority(ctx context.Context, agentID string, taskID string, task func(ctx context.Context) error, highPriority bool) <-chan error {
	q.mu.Lock()
	defer q.mu.Unlock()

	queues := q.normal
	if highPriority {
		queues = q.high
	}

	queue, exists := queues[agentID]
	if !exists {
		queue = &agentQueue{
			tasks:   make(chan *queuedTask, 100),
			running: false,
		}
		queues[agentID] = queue
	}

	qt := &queuedTask{
		task:    task,
		ctx:     ctx,
		done:    make(chan error, 1),
		taskID:  taskID,
		agentID: agentID,
	}

	if !queue.running {
		queue.running = true
		go q.processQueue(agentID, highPriority)
	}

	select {
	case queue.tasks <- qt:
		return qt.done
	default:
		qt.done <- ErrQueueFull
		close(qt.done)
		return qt.done
	}
}

func (q *PrioritySessionQueue) processQueue(agentID string, highPriority bool) {
	q.mu.Lock()
	queues := q.normal
	if highPriority {
		queues = q.high
	}
	queue, exists := queues[agentID]
	if !exists {
		q.mu.Unlock()
		return
	}
	q.mu.Unlock()

	for {
		// First check high priority queue
		if highPriority {
			select {
			case task := <-queue.tasks:
				q.executeTask(task, agentID)
				continue
			default:
			}
		}

		// Then check normal queue
		q.mu.Lock()
		normalQueue, normalExists := q.normal[agentID]
		q.mu.Unlock()

		if normalExists && len(normalQueue.tasks) > 0 && !highPriority {
			select {
			case task := <-normalQueue.tasks:
				q.executeTask(task, agentID)
				continue
			default:
			}
		}

		// No more tasks
		q.mu.Lock()
		if len(queue.tasks) == 0 {
			queue.running = false
			q.mu.Unlock()
			return
		}
		q.mu.Unlock()
	}
}

func (q *PrioritySessionQueue) executeTask(task *queuedTask, agentID string) {
	start := time.Now()
	err := task.task(task.ctx)
	duration := time.Since(start)

	logger.DebugCF("delegation", "Priority task executed", map[string]any{
		"task_id":     task.taskID,
		"agent_id":    agentID,
		"duration_ms": duration.Milliseconds(),
		"error":       err != nil,
	})

	task.done <- err
	close(task.done)
}
