package agent

import (
	"os"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/compactor"
	"github.com/RealityLink-Tech/MoonHub/pkg/config"
)

func TestNewAgentInstance_CompassorWhenEnabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-compactor-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				Model:     "test-model",
				MaxTokens: 8192,
			},
		},
		Compactor: config.DefaultCompactorConfig(),
	}
	cfg.Compactor.Enabled = true

	ag := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &mockProvider{})
	if ag.Compactor == nil {
		t.Fatal("expected non-nil Compactor when cfg.Compactor.Enabled and provider set")
	}
	c, ok := ag.Compactor.(*compactor.Compactor)
	if !ok {
		t.Fatalf("Compactor type = %T, want *compactor.Compactor", ag.Compactor)
	}
	if !c.GetConfig().Enabled {
		t.Fatal("compactor internal Config.Enabled must be true")
	}
	if ag.CompactorTriggerTokenPercent < 1 || ag.CompactorTriggerTokenPercent > 95 {
		t.Fatalf("CompactorTriggerTokenPercent=%d out of expected range", ag.CompactorTriggerTokenPercent)
	}
	if err := ag.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewAgentInstance_NoCompactorWhenDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-compactor-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				Model:     "test-model",
				MaxTokens: 8192,
			},
		},
		Compactor: config.DefaultCompactorConfig(),
	}
	cfg.Compactor.Enabled = false

	ag := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &mockProvider{})
	if ag.Compactor != nil {
		t.Fatal("expected nil Compactor when disabled")
	}
}

func TestNewAgentInstance_CompassorTriggerPercentClamped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-compactor-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				Model:     "test-model",
				MaxTokens: 8192,
			},
		},
		Compactor: config.DefaultCompactorConfig(),
	}
	cfg.Compactor.Enabled = true
	cfg.Compactor.TriggerTokenPercent = 0

	ag := NewAgentInstance(nil, &cfg.Agents.Defaults, cfg, &mockProvider{})
	if ag.CompactorTriggerTokenPercent != 70 {
		t.Fatalf("CompactorTriggerTokenPercent=%d want 70 for zero input", ag.CompactorTriggerTokenPercent)
	}
	_ = ag.Close()

	cfg2 := *cfg
	cfg2.Compactor.TriggerTokenPercent = 100
	ag2 := NewAgentInstance(nil, &cfg2.Agents.Defaults, &cfg2, &mockProvider{})
	if ag2.CompactorTriggerTokenPercent != 95 {
		t.Fatalf("CompactorTriggerTokenPercent=%d want 95 for 100 input", ag2.CompactorTriggerTokenPercent)
	}
	_ = ag2.Close()
}
