package tools

import (
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/shield"
)

func TestNewShieldEvaluatorFromEngine_Nil(t *testing.T) {
	t.Parallel()
	if NewShieldEvaluatorFromEngine(nil) != nil {
		t.Fatal("expected nil adapter for nil engine")
	}
}

func TestShieldAdapterDelegatesEvaluate(t *testing.T) {
	t.Parallel()
	eng := shield.NewEngineWithDefaults()
	ad := NewShieldEvaluatorFromEngine(eng)
	if ad == nil || !ad.IsActive() {
		t.Fatal("expected active adapter")
	}
	d := ad.Evaluate(ShieldEvent{
		Scope:    ScopeToolCall,
		ToolName: "nonexistent_tool_for_shield_adapter_test",
	})
	if d.Action != ActionLog {
		t.Fatalf("expected ActionLog for no match, got %q", d.Action)
	}
}
