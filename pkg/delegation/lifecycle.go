package delegation

import (
	"context"
	"fmt"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// LifecycleManager manages the lifecycle of sub-agents
type LifecycleManager struct {
	store    DelegationStore
	config   DelegationConfig
	intercom *Intercom
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(store DelegationStore, config DelegationConfig, intercom *Intercom) *LifecycleManager {
	return &LifecycleManager{
		store:    store,
		config:   config,
		intercom: intercom,
	}
}

// Create creates a new sub-agent
func (m *LifecycleManager) Create(ctx context.Context, userID, label, rolePrompt string, keywords, tools []string) (*SubAgentRecord, error) {
	// Check max active limit
	active, err := m.store.GetActiveSubAgents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active agents: %w", err)
	}

	if len(active) >= m.config.MaxActivePerUser {
		// Try to dismiss oldest inactive agent
		if err := m.dismissOldestInactive(ctx, active); err != nil {
			return nil, fmt.Errorf("max active agents (%d) reached and failed to dismiss old agent", m.config.MaxActivePerUser)
		}
	}

	now := time.Now().UnixMilli()
	agent := &SubAgentRecord{
		ID:           generateID("agent"),
		UserID:       userID,
		Label:        label,
		RolePrompt:   rolePrompt,
		TaskKeywords: keywords,
		Tools:        tools,
		Status:       StatusActive,
		CreatedAt:    now,
		LastActiveAt: now,
		SessionKey:   fmt.Sprintf("subagent:%s", generateID("session")),
	}

	if err := m.store.SaveSubAgent(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to save sub-agent: %w", err)
	}

	// Emit event
	m.intercom.Emit(ctx, TopicAgentCreated, userID, map[string]any{
		"agent_id": agent.ID,
		"label":    agent.Label,
	})

	logger.InfoCF("delegation", "Created sub-agent", map[string]any{
		"agent_id": agent.ID,
		"label":    agent.Label,
		"user_id":  userID,
	})

	return agent, nil
}

// Get retrieves a sub-agent by ID
func (m *LifecycleManager) Get(ctx context.Context, id string) (*SubAgentRecord, error) {
	return m.store.GetSubAgent(ctx, id)
}

// ListActive lists all active sub-agents for a user
func (m *LifecycleManager) ListActive(ctx context.Context, userID string) ([]*SubAgentRecord, error) {
	return m.store.GetActiveSubAgents(ctx, userID)
}

// FindReusable finds a sub-agent that can be reused for the given keywords
func (m *LifecycleManager) FindReusable(ctx context.Context, userID string, keywords []string) (*SubAgentRecord, error) {
	agents, err := m.store.GetActiveSubAgents(ctx, userID)
	if err != nil {
		return nil, err
	}

	var bestMatch *SubAgentRecord
	bestScore := m.config.ReuseThreshold

	for _, agent := range agents {
		score := keywordOverlapScore(keywords, agent.TaskKeywords)
		if score > bestScore {
			bestScore = score
			bestMatch = agent
		}
	}

	if bestMatch != nil {
		logger.InfoCF("delegation", "Found reusable sub-agent", map[string]any{
			"agent_id": bestMatch.ID,
			"label":    bestMatch.Label,
			"score":    bestScore,
		})
	}

	return bestMatch, nil
}

// Suspend suspends a sub-agent (can be revived later)
func (m *LifecycleManager) Suspend(ctx context.Context, id string) error {
	agent, err := m.store.GetSubAgent(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("sub-agent not found: %s", id)
	}

	if agent.Status != StatusActive {
		return fmt.Errorf("sub-agent is not active (status: %s)", agent.Status)
	}

	if err := m.store.UpdateSubAgent(ctx, id, map[string]any{
		"status":         string(StatusSuspended),
		"last_active_at": time.Now().UnixMilli(),
	}); err != nil {
		return err
	}

	m.intercom.Emit(ctx, TopicAgentSuspended, agent.UserID, map[string]any{
		"agent_id": id,
	})

	logger.InfoCF("delegation", "Suspended sub-agent", map[string]any{
		"agent_id": id,
	})

	return nil
}

// Dismiss dismisses a sub-agent (cannot be revived)
func (m *LifecycleManager) Dismiss(ctx context.Context, id string) error {
	agent, err := m.store.GetSubAgent(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("sub-agent not found: %s", id)
	}

	if err := m.store.UpdateSubAgent(ctx, id, map[string]any{
		"status":         string(StatusDismissed),
		"last_active_at": time.Now().UnixMilli(),
	}); err != nil {
		return err
	}

	m.intercom.Emit(ctx, TopicAgentDismissed, agent.UserID, map[string]any{
		"agent_id": id,
	})

	logger.InfoCF("delegation", "Dismissed sub-agent", map[string]any{
		"agent_id": id,
	})

	return nil
}

// Revive revives a suspended sub-agent
func (m *LifecycleManager) Revive(ctx context.Context, id string) error {
	agent, err := m.store.GetSubAgent(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("sub-agent not found: %s", id)
	}

	if agent.Status != StatusSuspended {
		return fmt.Errorf("sub-agent is not suspended (status: %s)", agent.Status)
	}

	// Check max active limit
	active, err := m.store.GetActiveSubAgents(ctx, agent.UserID)
	if err != nil {
		return err
	}

	if len(active) >= m.config.MaxActivePerUser {
		return fmt.Errorf("max active agents (%d) reached", m.config.MaxActivePerUser)
	}

	if err := m.store.UpdateSubAgent(ctx, id, map[string]any{
		"status":         string(StatusActive),
		"last_active_at": time.Now().UnixMilli(),
	}); err != nil {
		return err
	}

	m.intercom.Emit(ctx, TopicAgentRevived, agent.UserID, map[string]any{
		"agent_id": id,
	})

	logger.InfoCF("delegation", "Revived sub-agent", map[string]any{
		"agent_id": id,
	})

	return nil
}

// Kill immediately kills a sub-agent and removes it
func (m *LifecycleManager) Kill(ctx context.Context, id string) error {
	agent, err := m.store.GetSubAgent(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("sub-agent not found: %s", id)
	}

	if err := m.store.DeleteSubAgent(ctx, id); err != nil {
		return err
	}

	m.intercom.Emit(ctx, TopicAgentDismissed, agent.UserID, map[string]any{
		"agent_id": id,
		"killed":   true,
	})

	logger.InfoCF("delegation", "Killed sub-agent", map[string]any{
		"agent_id": id,
	})

	return nil
}

// RecordTaskResult records the result of a task execution
func (m *LifecycleManager) RecordTaskResult(ctx context.Context, id string, success bool, durationMs int64) error {
	agent, err := m.store.GetSubAgent(ctx, id)
	if err != nil || agent == nil {
		return err
	}

	now := time.Now().UnixMilli()
	updates := map[string]any{
		"last_active_at": now,
	}

	// Update completed tasks count
	updates["completed_tasks"] = agent.CompletedTasks + 1

	// Update success rate (exponential moving average)
	if agent.CompletedTasks == 0 {
		if success {
			updates["success_rate"] = 1.0
		} else {
			updates["success_rate"] = 0.0
		}
	} else {
		alpha := 0.2 // Smoothing factor
		var newRate float64
		if success {
			newRate = agent.SuccessRate*(1-alpha) + 1.0*alpha
		} else {
			newRate = agent.SuccessRate * (1 - alpha)
		}
		updates["success_rate"] = newRate
	}

	if err := m.store.UpdateSubAgent(ctx, id, updates); err != nil {
		return err
	}

	// If success rate drops too low, auto-dismiss
	agent.SuccessRate = updates["success_rate"].(float64)
	if agent.CompletedTasks >= 5 && agent.SuccessRate < 0.2 {
		logger.WarnCF("delegation", "Auto-dismissing low-performing sub-agent", map[string]any{
			"agent_id":     id,
			"success_rate": agent.SuccessRate,
		})
		return m.Dismiss(ctx, id)
	}

	return nil
}

// SaveMessage saves a message to a sub-agent's history
func (m *LifecycleManager) SaveMessage(ctx context.Context, subAgentID, role, content string) error {
	msg := &SubAgentMessage{
		ID:         generateID("msg"),
		SubAgentID: subAgentID,
		Role:       role,
		Content:    content,
		CreatedAt:  time.Now().UnixMilli(),
	}
	return m.store.SaveSubAgentMessage(ctx, msg)
}

// GetMessages retrieves messages for a sub-agent
func (m *LifecycleManager) GetMessages(ctx context.Context, subAgentID string, limit int) ([]*SubAgentMessage, error) {
	return m.store.GetSubAgentMessages(ctx, subAgentID, limit)
}

// dismissOldestInactive dismisses the oldest inactive agent
func (m *LifecycleManager) dismissOldestInactive(ctx context.Context, agents []*SubAgentRecord) error {
	if len(agents) == 0 {
		return nil
	}

	// Find oldest by last_active_at
	oldest := agents[0]
	for _, a := range agents[1:] {
		if a.LastActiveAt < oldest.LastActiveAt {
			oldest = a
		}
	}

	return m.Dismiss(ctx, oldest.ID)
}

// GetSummary creates a summary of a sub-agent
func (m *LifecycleManager) GetSummary(agent *SubAgentRecord) SubAgentSummary {
	return SubAgentSummary{
		ID:             agent.ID,
		Label:          agent.Label,
		Status:         agent.Status,
		CompletedTasks: agent.CompletedTasks,
		SuccessRate:    agent.SuccessRate,
		LastActiveAt:   agent.LastActiveAt,
		CreatedAt:      agent.CreatedAt,
	}
}

// CleanupOldData removes old data based on retention policy
func (m *LifecycleManager) CleanupOldData(ctx context.Context) error {
	return m.store.CleanupOldData(ctx, m.config.RetentionDays)
}
