package agent

import (
	"os"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

// TestAgentInstance_delegationEnabled_wiresTools verifies runtime integration:
// enabled delegation attaches DelegationIntegration and registers delegate_* tools.
func TestAgentInstance_delegationEnabled_wiresTools(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "delegation-agent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "openai/test-model",
				MaxToolIterations: 10,
			},
		},
		Delegation: config.DelegationConfig{
			Enabled:              true,
			DefaultSubAgentTools: []string{"read_file", "list_dir"},
			MaxActivePerUser:     8,
			MaxConcurrentTasks:   2,
			RetentionDays:        7,
			ReuseThreshold:       0.6,
		},
	}

	ag := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &simpleMockProvider{response: "ok"})
	defer func() {
		if ag.Delegation != nil {
			_ = ag.Delegation.Close()
		}
	}()

	if ag.Delegation == nil || !ag.Delegation.IsEnabled() {
		t.Fatalf("expected Delegation enabled, got %+v", ag.Delegation)
	}
	for _, name := range []string{
		"delegate_task",
		"delegate_tasks",
		"delegate_background",
		"delegate_to_existing",
		"list_sub_agents",
		"manage_sub_agent",
		"manage_template",
		"confirm_task",
	} {
		if _, ok := ag.Tools.Get(name); !ok {
			t.Errorf("tool %q not registered", name)
		}
	}
}
