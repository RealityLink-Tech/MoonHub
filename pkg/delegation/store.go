package delegation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// DelegationStore defines the interface for delegation persistence.
//
//nolint:interfacebloat // Store facade mirrors persistence surface; splitting would scatter callers.
type DelegationStore interface {
	// Sub-agent operations
	SaveSubAgent(ctx context.Context, agent *SubAgentRecord) error
	GetSubAgent(ctx context.Context, id string) (*SubAgentRecord, error)
	GetActiveSubAgents(ctx context.Context, userID string) ([]*SubAgentRecord, error)
	UpdateSubAgent(ctx context.Context, id string, updates map[string]any) error
	DeleteSubAgent(ctx context.Context, id string) error

	// Role template operations
	SaveRoleTemplate(ctx context.Context, template *RoleTemplate) error
	GetRoleTemplates(ctx context.Context, userID string) ([]*RoleTemplate, error)
	FindBestMatchTemplate(ctx context.Context, userID string, keywords []string, threshold float64) (*RoleTemplate, error)
	DeleteRoleTemplate(ctx context.Context, id string) error

	// Background task operations
	SaveBackgroundTask(ctx context.Context, task *BackgroundTaskRecord) error
	UpdateBackgroundTask(ctx context.Context, id string, updates map[string]any) error
	GetUndeliveredTasks(ctx context.Context, userID string) ([]*BackgroundTaskRecord, error)
	MarkTaskDelivered(ctx context.Context, id string) error

	// Task metrics operations
	SaveTaskMetric(ctx context.Context, metric *TaskMetricRecord) error
	GetTaskMetrics(ctx context.Context, userID string, category TaskCategory, limit int) ([]*TaskMetricRecord, error)

	// Sub-agent message operations
	SaveSubAgentMessage(ctx context.Context, msg *SubAgentMessage) error
	GetSubAgentMessages(ctx context.Context, subAgentID string, limit int) ([]*SubAgentMessage, error)

	// Maintenance
	CleanupOldData(ctx context.Context, retentionDays int) error
	Close() error
}

// SQLiteStore implements DelegationStore using SQLite
type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteStore creates a new SQLite delegation store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open delegation database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema() error {
	schema := `
	-- Sub-agents table
	CREATE TABLE IF NOT EXISTS sub_agents (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		label TEXT NOT NULL,
		role_prompt TEXT NOT NULL,
		task_keywords TEXT NOT NULL DEFAULT '[]',
		tools TEXT NOT NULL DEFAULT '[]',
		status TEXT NOT NULL DEFAULT 'active',
		created_at INTEGER NOT NULL,
		last_active_at INTEGER NOT NULL,
		completed_tasks INTEGER NOT NULL DEFAULT 0,
		success_rate REAL NOT NULL DEFAULT 0,
		session_key TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_sub_agents_user_status ON sub_agents (user_id, status);

	-- Role templates table
	CREATE TABLE IF NOT EXISTS role_templates (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		role_prompt TEXT NOT NULL,
		task_keywords TEXT NOT NULL DEFAULT '[]',
		tools TEXT NOT NULL DEFAULT '[]',
		category TEXT NOT NULL DEFAULT 'general',
		success_count INTEGER NOT NULL DEFAULT 0,
		failure_count INTEGER NOT NULL DEFAULT 0,
		avg_duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		last_used_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_role_templates_user ON role_templates (user_id);

	-- Background tasks table
	CREATE TABLE IF NOT EXISTS background_tasks (
		id TEXT PRIMARY KEY,
		sub_agent_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		task TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'general',
		status TEXT NOT NULL DEFAULT 'pending',
		result TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL,
		completed_at INTEGER NOT NULL DEFAULT 0,
		delivered INTEGER NOT NULL DEFAULT 0,
		origin_channel TEXT NOT NULL DEFAULT '',
		origin_chat_id TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_bg_tasks_user_delivered ON background_tasks (user_id, delivered);
	CREATE INDEX IF NOT EXISTS idx_bg_tasks_status ON background_tasks (status);

	-- Task metrics table
	CREATE TABLE IF NOT EXISTS task_metrics (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		task_category TEXT NOT NULL,
		task_hint TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		success INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_task_metrics_user_category ON task_metrics (user_id, task_category);

	-- Sub-agent messages table
	CREATE TABLE IF NOT EXISTS sub_agent_messages (
		id TEXT PRIMARY KEY,
		sub_agent_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_messages_subagent ON sub_agent_messages (sub_agent_id, created_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveSubAgent saves a sub-agent record
func (s *SQLiteStore) SaveSubAgent(ctx context.Context, agent *SubAgentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keywordsJSON := strings.Join(agent.TaskKeywords, "||")
	toolsJSON := strings.Join(agent.Tools, "||")

	query := `
		INSERT OR REPLACE INTO sub_agents
		(id, user_id, label, role_prompt, task_keywords, tools, status, created_at, last_active_at, completed_tasks, success_rate, session_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		agent.ID, agent.UserID, agent.Label, agent.RolePrompt,
		keywordsJSON, toolsJSON, agent.Status,
		agent.CreatedAt, agent.LastActiveAt, agent.CompletedTasks, agent.SuccessRate, agent.SessionKey,
	)

	if err != nil {
		logger.ErrorCF("delegation", "Failed to save sub-agent", map[string]any{
			"id":    agent.ID,
			"error": err.Error(),
		})
	}
	return err
}

// GetSubAgent retrieves a sub-agent by ID
func (s *SQLiteStore) GetSubAgent(ctx context.Context, id string) (*SubAgentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, user_id, label, role_prompt, task_keywords, tools, status,
		       created_at, last_active_at, completed_tasks, success_rate, session_key
		FROM sub_agents WHERE id = ?
	`

	var agent SubAgentRecord
	var keywordsStr, toolsStr string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&agent.ID, &agent.UserID, &agent.Label, &agent.RolePrompt,
		&keywordsStr, &toolsStr, &agent.Status,
		&agent.CreatedAt, &agent.LastActiveAt, &agent.CompletedTasks, &agent.SuccessRate, &agent.SessionKey,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	agent.TaskKeywords = parseJSONArray(keywordsStr)
	agent.Tools = parseJSONArray(toolsStr)

	return &agent, nil
}

// GetActiveSubAgents retrieves all active sub-agents for a user.
//
//nolint:dupl // Same scan loop shape as GetRoleTemplates; different types/columns.
func (s *SQLiteStore) GetActiveSubAgents(ctx context.Context, userID string) ([]*SubAgentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, user_id, label, role_prompt, task_keywords, tools, status,
		       created_at, last_active_at, completed_tasks, success_rate, session_key
		FROM sub_agents WHERE user_id = ? AND status = 'active'
		ORDER BY last_active_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*SubAgentRecord
	for rows.Next() {
		var agent SubAgentRecord
		var keywordsStr, toolsStr string

		err := rows.Scan(
			&agent.ID, &agent.UserID, &agent.Label, &agent.RolePrompt,
			&keywordsStr, &toolsStr, &agent.Status,
			&agent.CreatedAt, &agent.LastActiveAt, &agent.CompletedTasks, &agent.SuccessRate, &agent.SessionKey,
		)
		if err != nil {
			return nil, err
		}

		agent.TaskKeywords = parseJSONArray(keywordsStr)
		agent.Tools = parseJSONArray(toolsStr)
		agents = append(agents, &agent)
	}

	return agents, rows.Err()
}

// UpdateSubAgent updates specific fields of a sub-agent
func (s *SQLiteStore) UpdateSubAgent(ctx context.Context, id string, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(updates) == 0 {
		return nil
	}

	var setClauses []string
	var args []any

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE sub_agents SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteSubAgent deletes a sub-agent
func (s *SQLiteStore) DeleteSubAgent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM sub_agents WHERE id = ?", id)
	return err
}

// SaveRoleTemplate saves a role template
func (s *SQLiteStore) SaveRoleTemplate(ctx context.Context, template *RoleTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keywordsJSON := strings.Join(template.TaskKeywords, "||")
	toolsJSON := strings.Join(template.Tools, "||")

	query := `
		INSERT OR REPLACE INTO role_templates
		(id, user_id, name, role_prompt, task_keywords, tools, category, success_count, failure_count, avg_duration_ms, created_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		template.ID, template.UserID, template.Name, template.RolePrompt,
		keywordsJSON, toolsJSON, template.Category,
		template.SuccessCount, template.FailureCount, template.AvgDurationMs,
		template.CreatedAt, template.LastUsedAt,
	)

	return err
}

// GetRoleTemplates retrieves all role templates for a user.
//
//nolint:dupl // Same scan loop shape as GetActiveSubAgents; different types/columns.
func (s *SQLiteStore) GetRoleTemplates(ctx context.Context, userID string) ([]*RoleTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, user_id, name, role_prompt, task_keywords, tools, category,
		       success_count, failure_count, avg_duration_ms, created_at, last_used_at
		FROM role_templates WHERE user_id = ?
		ORDER BY last_used_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*RoleTemplate
	for rows.Next() {
		var t RoleTemplate
		var keywordsStr, toolsStr string

		err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.RolePrompt,
			&keywordsStr, &toolsStr, &t.Category,
			&t.SuccessCount, &t.FailureCount, &t.AvgDurationMs,
			&t.CreatedAt, &t.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}

		t.TaskKeywords = parseJSONArray(keywordsStr)
		t.Tools = parseJSONArray(toolsStr)
		templates = append(templates, &t)
	}

	return templates, rows.Err()
}

// FindBestMatchTemplate finds the best matching template based on keyword overlap
func (s *SQLiteStore) FindBestMatchTemplate(ctx context.Context, userID string, keywords []string, threshold float64) (*RoleTemplate, error) {
	templates, err := s.GetRoleTemplates(ctx, userID)
	if err != nil {
		return nil, err
	}

	var bestMatch *RoleTemplate
	bestScore := threshold

	for _, t := range templates {
		score := keywordOverlapScore(keywords, t.TaskKeywords)
		if score > bestScore {
			bestScore = score
			bestMatch = t
		}
	}

	return bestMatch, nil
}

// DeleteRoleTemplate deletes a role template
func (s *SQLiteStore) DeleteRoleTemplate(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM role_templates WHERE id = ?", id)
	return err
}

// SaveBackgroundTask saves a background task
func (s *SQLiteStore) SaveBackgroundTask(ctx context.Context, task *BackgroundTaskRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT OR REPLACE INTO background_tasks
		(id, sub_agent_id, user_id, task, category, status, result, created_at, completed_at, delivered, origin_channel, origin_chat_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	delivered := 0
	if task.Delivered {
		delivered = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.SubAgentID, task.UserID, task.Task, task.Category,
		task.Status, task.Result, task.CreatedAt, task.CompletedAt, delivered,
		task.OriginChannel, task.OriginChatID,
	)

	return err
}

// UpdateBackgroundTask updates specific fields of a background task
func (s *SQLiteStore) UpdateBackgroundTask(ctx context.Context, id string, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(updates) == 0 {
		return nil
	}

	var setClauses []string
	var args []any

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE background_tasks SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetUndeliveredTasks retrieves all undelivered completed tasks for a user
func (s *SQLiteStore) GetUndeliveredTasks(ctx context.Context, userID string) ([]*BackgroundTaskRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, sub_agent_id, user_id, task, category, status, result,
		       created_at, completed_at, delivered, origin_channel, origin_chat_id
		FROM background_tasks
		WHERE user_id = ? AND delivered = 0 AND status IN ('completed', 'failed')
		ORDER BY completed_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*BackgroundTaskRecord
	for rows.Next() {
		var t BackgroundTaskRecord
		var delivered int

		err := rows.Scan(
			&t.ID, &t.SubAgentID, &t.UserID, &t.Task, &t.Category,
			&t.Status, &t.Result, &t.CreatedAt, &t.CompletedAt, &delivered,
			&t.OriginChannel, &t.OriginChatID,
		)
		if err != nil {
			return nil, err
		}

		t.Delivered = delivered == 1
		tasks = append(tasks, &t)
	}

	return tasks, rows.Err()
}

// MarkTaskDelivered marks a task as delivered
func (s *SQLiteStore) MarkTaskDelivered(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "UPDATE background_tasks SET delivered = 1 WHERE id = ?", id)
	return err
}

// SaveTaskMetric saves a task metric
func (s *SQLiteStore) SaveTaskMetric(ctx context.Context, metric *TaskMetricRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	success := 0
	if metric.Success {
		success = 1
	}

	query := `
		INSERT INTO task_metrics (id, user_id, task_category, task_hint, duration_ms, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		metric.ID, metric.UserID, metric.TaskCategory, metric.TaskHint,
		metric.DurationMs, success, metric.CreatedAt,
	)

	return err
}

// GetTaskMetrics retrieves recent task metrics for a user and category
func (s *SQLiteStore) GetTaskMetrics(ctx context.Context, userID string, category TaskCategory, limit int) ([]*TaskMetricRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, user_id, task_category, task_hint, duration_ms, success, created_at
		FROM task_metrics
		WHERE user_id = ? AND task_category = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*TaskMetricRecord
	for rows.Next() {
		var m TaskMetricRecord
		var success int

		err := rows.Scan(
			&m.ID, &m.UserID, &m.TaskCategory, &m.TaskHint,
			&m.DurationMs, &success, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		m.Success = success == 1
		metrics = append(metrics, &m)
	}

	return metrics, rows.Err()
}

// SaveSubAgentMessage saves a message to a sub-agent's history
func (s *SQLiteStore) SaveSubAgentMessage(ctx context.Context, msg *SubAgentMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO sub_agent_messages (id, sub_agent_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		msg.ID, msg.SubAgentID, msg.Role, msg.Content, msg.CreatedAt,
	)

	return err
}

// GetSubAgentMessages retrieves messages for a sub-agent
func (s *SQLiteStore) GetSubAgentMessages(ctx context.Context, subAgentID string, limit int) ([]*SubAgentMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, sub_agent_id, role, content, created_at
		FROM sub_agent_messages
		WHERE sub_agent_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, subAgentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*SubAgentMessage
	for rows.Next() {
		var m SubAgentMessage
		err := rows.Scan(&m.ID, &m.SubAgentID, &m.Role, &m.Content, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}

// CleanupOldData removes old records based on retention period
func (s *SQLiteStore) CleanupOldData(ctx context.Context, retentionDays int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays).UnixMilli()

	// Clean up dismissed agents
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sub_agents WHERE status = 'dismissed' AND last_active_at < ?",
		cutoff,
	)
	if err != nil {
		return err
	}

	// Clean up old task metrics
	_, err = s.db.ExecContext(ctx,
		"DELETE FROM task_metrics WHERE created_at < ?",
		cutoff,
	)
	if err != nil {
		return err
	}

	// Clean up delivered background tasks
	_, err = s.db.ExecContext(ctx,
		"DELETE FROM background_tasks WHERE delivered = 1 AND completed_at < ?",
		cutoff,
	)
	if err != nil {
		return err
	}

	// Clean up old messages
	_, err = s.db.ExecContext(ctx,
		"DELETE FROM sub_agent_messages WHERE created_at < ?",
		cutoff,
	)

	logger.InfoCF("delegation", "Cleaned up old delegation data", map[string]any{
		"retention_days": retentionDays,
	})

	return err
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Helper functions

func parseJSONArray(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	return strings.Split(s, "||")
}

func keywordOverlapScore(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0
	}

	// Normalize keywords to lowercase
	set1 := make(map[string]bool)
	for _, k := range keywords1 {
		set1[strings.ToLower(k)] = true
	}

	matches := 0
	for _, k := range keywords2 {
		if set1[strings.ToLower(k)] {
			matches++
		}
	}

	// Jaccard-like similarity
	return float64(matches) / float64(max(len(keywords1), len(keywords2)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
