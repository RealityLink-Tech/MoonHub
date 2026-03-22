package agent

import (
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/routing"
)

func TestConvertTierBoundaries_explicitZeroSimpleModerate(t *testing.T) {
	z := 0.0
	cfg := &config.TierBoundariesConfig{SimpleModerate: &z}
	m := convertTierBoundaries(cfg)
	if m == nil {
		t.Fatal("expected boundaries with sm=0")
	}
	if b := m[routing.TierSimple]; b.Max != 0 {
		t.Errorf("simple max: got %v want 0", b.Max)
	}
}

func TestConvertTierBoundaries_invalidRevertsToNil(t *testing.T) {
	sm, mc, cr := -0.05, 0.35, 0.15 // cr < mc
	cfg := &config.TierBoundariesConfig{
		SimpleModerate:   &sm,
		ModerateComplex:  &mc,
		ComplexReasoning: &cr,
	}
	if m := convertTierBoundaries(cfg); m != nil {
		t.Fatalf("expected nil for invalid cutpoints, got %v", m)
	}
}

func TestConvertTierBoundaries_nilConfig(t *testing.T) {
	if m := convertTierBoundaries(nil); m != nil {
		t.Fatalf("want nil, got %v", m)
	}
	empty := &config.TierBoundariesConfig{}
	if m := convertTierBoundaries(empty); m != nil {
		t.Fatalf("empty config should mean defaults (nil map), got %v", m)
	}
}

func TestConvertTierBoundaries_partialPointers(t *testing.T) {
	mc := 0.2
	cfg := &config.TierBoundariesConfig{ModerateComplex: &mc}
	m := convertTierBoundaries(cfg)
	if m == nil {
		t.Fatal("expected map")
	}
	// sm default -0.05, cr default 0.35
	if err := routing.ValidateTierCutpoints(-0.05, 0.2, 0.35); err != nil {
		t.Fatalf("built map should be valid: %v", err)
	}
}
