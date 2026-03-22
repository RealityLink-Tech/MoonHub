// Package agent provides the agent loop and integration for MoonHub
package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/delegation"
	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

func delegationConfigFromApp(c config.DelegationConfig) delegation.DelegationConfig {
	return delegation.DelegationConfig{
		Enabled:              c.Enabled,
		DefaultSubAgentTools: c.DefaultSubAgentTools,
		MaxActivePerUser:     c.MaxActivePerUser,
		MaxConcurrentTasks:   c.MaxConcurrentTasks,
		RetentionDays:        c.RetentionDays,
		ReuseThreshold:       c.ReuseThreshold,
	}
}

// DelegationIntegration handles the integration of the delegation system with the agent loop
type DelegationIntegration struct {
	delegation *delegation.DelegationSystem
	workspace  string
}

// NewDelegationIntegration creates a new delegation integration
func NewDelegationIntegration(
	workspace string,
	provider providers.LLMProvider,
	model string,
	cfg config.DelegationConfig,
) (*DelegationIntegration, error) {
	if !cfg.Enabled {
		return &DelegationIntegration{}, nil
	}

	// Ensure workspace exists
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create delegation system
	dbPath := delegation.GetDBPath(workspace)
	dc := delegationConfigFromApp(cfg)
	ds, err := delegation.NewDelegationSystem(dbPath, dc, provider, model, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegation system: %w", err)
	}

	logger.InfoCF("agent", "Delegation system initialized", map[string]any{
		"workspace": workspace,
		"db_path":   dbPath,
	})

	return &DelegationIntegration{
		delegation: ds,
		workspace:  workspace,
	}, nil
}

// RegisterTools registers delegation tools to the tool registry
func (di *DelegationIntegration) RegisterTools(registry *tools.ToolRegistry) {
	if di.delegation == nil {
		return
	}
	di.delegation.RegisterTools(registry)
}

// GetUndeliveredResults gets background task results to inject into conversation
func (di *DelegationIntegration) GetUndeliveredResults(ctx context.Context, userID string) ([]*delegation.BackgroundTaskRecord, error) {
	if di.delegation == nil {
		return nil, nil
	}
	return di.delegation.GetUndeliveredResults(ctx, userID)
}

// MarkDelivered marks a task as delivered
func (di *DelegationIntegration) MarkDelivered(ctx context.Context, taskID string) error {
	if di.delegation == nil {
		return nil
	}
	return di.delegation.MarkDelivered(ctx, taskID)
}

// Close closes the delegation system
func (di *DelegationIntegration) Close() error {
	if di.delegation == nil {
		return nil
	}
	return di.delegation.Close()
}

// IsEnabled returns whether delegation is enabled
func (di *DelegationIntegration) IsEnabled() bool {
	return di.delegation != nil
}

// Stats returns statistics about the delegation system
func (di *DelegationIntegration) Stats() map[string]any {
	if di.delegation == nil {
		return map[string]any{"enabled": false}
	}
	return di.delegation.Stats()
}

// FormatResultsForContext formats background task results for LLM context injection
func FormatResultsForContext(records []*delegation.BackgroundTaskRecord) string {
	if len(records) == 0 {
		return ""
	}

	var results string
	results = "[System: Background Task Results]\n\n"
	results += "The following background tasks have completed:\n\n"

	for _, r := range records {
		results += delegation.FormatTaskResultForLLM(r) + "\n\n"
	}

	results += "These results are from tasks you delegated earlier. Acknowledge them and continue the conversation naturally."

	return results
}

// RegisterDelegationTools registers delegation tools to an agent's tool registry
// This is the main integration point for the delegation system
func RegisterDelegationTools(
	agent *AgentInstance,
	provider providers.LLMProvider,
	workspace string,
	cfg config.DelegationConfig,
) (*DelegationIntegration, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Create integration
	di, err := NewDelegationIntegration(workspace, provider, agent.Model, cfg)
	if err != nil {
		return nil, err
	}

	// Register tools
	if di != nil {
		di.RegisterTools(agent.Tools)
	}

	return di, nil
}

// InjectBackgroundTaskResults injects completed background task results into the conversation
// This should be called at the start of ProcessMessage
func InjectBackgroundTaskResults(
	ctx context.Context,
	di *DelegationIntegration,
	userID string,
) string {
	if di == nil || !di.IsEnabled() {
		return ""
	}

	results, err := di.GetUndeliveredResults(ctx, userID)
	if err != nil {
		logger.WarnCF("agent", "Failed to get undelivered background task results", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	if len(results) == 0 {
		return ""
	}

	// Mark as delivered
	for _, r := range results {
		di.MarkDelivered(ctx, r.ID)
	}

	return FormatResultsForContext(results)
}
