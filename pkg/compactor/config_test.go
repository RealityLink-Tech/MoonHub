// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("Default config should have Enabled = true")
	}
	if cfg.TriggerTokenPercent != 70 {
		t.Errorf("Default TriggerTokenPercent = %d, want 70", cfg.TriggerTokenPercent)
	}
	if cfg.KeepRecent != 10 {
		t.Errorf("Default KeepRecent = %d, want 10", cfg.KeepRecent)
	}
	if cfg.DedupSimilarityThreshold != 0.6 {
		t.Errorf("Default DedupSimilarityThreshold = %f, want 0.6", cfg.DedupSimilarityThreshold)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		check  func(Config) bool
	}{
		{
			name:   "default config is valid",
			config: DefaultConfig(),
			check: func(c Config) bool {
				return c.TriggerTokenPercent > 0 && c.TriggerTokenPercent <= 95
			},
		},
		{
			name:   "zero TriggerTokenPercent gets default",
			config: Config{TriggerTokenPercent: 0},
			check: func(c Config) bool {
				return c.TriggerTokenPercent == 70
			},
		},
		{
			name:   "TriggerTokenPercent > 95 is capped",
			config: Config{TriggerTokenPercent: 100},
			check: func(c Config) bool {
				return c.TriggerTokenPercent == 95
			},
		},
		{
			name:   "negative TriggerTokenPercent gets default",
			config: Config{TriggerTokenPercent: -1},
			check: func(c Config) bool {
				return c.TriggerTokenPercent == 70
			},
		},
		{
			name:   "zero KeepRecent gets default",
			config: Config{KeepRecent: 0},
			check: func(c Config) bool {
				return c.KeepRecent == 10
			},
		},
		{
			name:   "zero tier budgets get defaults",
			config: Config{TierBudgets: TierBudgets{}},
			check: func(c Config) bool {
				return c.TierBudgets.L0 == 200 && c.TierBudgets.L1 == 1000 && c.TierBudgets.L2 == 3000
			},
		},
		{
			name:   "zero DedupSimilarityThreshold gets default",
			config: Config{DedupSimilarityThreshold: 0},
			check: func(c Config) bool {
				return c.DedupSimilarityThreshold == 0.6
			},
		},
		{
			name:   "DedupSimilarityThreshold > 1 is capped",
			config: Config{DedupSimilarityThreshold: 1.5},
			check: func(c Config) bool {
				return c.DedupSimilarityThreshold == 1.0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config
			cfg.Validate()
			if !tt.check(cfg) {
				t.Errorf("Config validation check failed")
			}
		})
	}
}

func TestConfig_ShouldTrigger(t *testing.T) {
	cfg := Config{
		Enabled:             true,
		TriggerTokenPercent: 70,
	}

	tests := []struct {
		currentTokens int
		contextWindow int
		want          bool
	}{
		{currentTokens: 50, contextWindow: 100, want: false}, // 50% < 70%
		{currentTokens: 70, contextWindow: 100, want: true},  // 70% >= 70%
		{currentTokens: 80, contextWindow: 100, want: true},  // 80% >= 70%
		{currentTokens: 100, contextWindow: 100, want: true}, // 100% >= 70%
	}

	for _, tt := range tests {
		got := cfg.ShouldTrigger(tt.currentTokens, tt.contextWindow)
		if got != tt.want {
			t.Errorf("ShouldTrigger(%d, %d) = %v, want %v",
				tt.currentTokens, tt.contextWindow, got, tt.want)
		}
	}
}

func TestConfig_ShouldTrigger_Disabled(t *testing.T) {
	cfg := Config{
		Enabled:             false,
		TriggerTokenPercent: 70,
	}

	// Should never trigger when disabled
	if cfg.ShouldTrigger(100, 100) {
		t.Error("ShouldTrigger should return false when disabled")
	}
}

func TestConfig_GetKeepRecentCount(t *testing.T) {
	tests := []struct {
		keepRecent int
		want       int
	}{
		{keepRecent: 5, want: 5},
		{keepRecent: 20, want: 20},
		{keepRecent: 0, want: 10},  // default
		{keepRecent: -1, want: 10}, // default
	}

	for _, tt := range tests {
		cfg := Config{KeepRecent: tt.keepRecent}
		got := cfg.GetKeepRecentCount()
		if got != tt.want {
			t.Errorf("GetKeepRecentCount() with KeepRecent=%d = %d, want %d",
				tt.keepRecent, got, tt.want)
		}
	}
}

func TestConfig_GetTierBudgets(t *testing.T) {
	tests := []struct {
		name    string
		budgets TierBudgets
		wantL0  int
		wantL1  int
		wantL2  int
	}{
		{
			name:    "custom budgets",
			budgets: TierBudgets{L0: 100, L1: 500, L2: 2000},
			wantL0:  100,
			wantL1:  500,
			wantL2:  2000,
		},
		{
			name:    "zero budgets get defaults",
			budgets: TierBudgets{L0: 0, L1: 0, L2: 0},
			wantL0:  200,
			wantL1:  1000,
			wantL2:  3000,
		},
		{
			name:    "partial zero budgets",
			budgets: TierBudgets{L0: 50, L1: 0, L2: 5000},
			wantL0:  50,
			wantL1:  1000, // default
			wantL2:  5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{TierBudgets: tt.budgets}
			got := cfg.GetTierBudgets()

			if got.L0 != tt.wantL0 || got.L1 != tt.wantL1 || got.L2 != tt.wantL2 {
				t.Errorf("GetTierBudgets() = {%d, %d, %d}, want {%d, %d, %d}",
					got.L0, got.L1, got.L2, tt.wantL0, tt.wantL1, tt.wantL2)
			}
		})
	}
}

func TestCompactionMetrics(t *testing.T) {
	metrics := CompactionMetrics{
		MessagesBefore:     100,
		MessagesSummarized: 80,
		MessagesKept:       10,
		TokensBefore:       5000,
		TokensAfter:        500,
		CompressionRatio:   0.1,
		DedupGroupsRemoved: 5,
		RulesApplied:       []string{"CJK", "Emoji", "Tables"},
		DurationMs:         1500,
	}

	// Verify fields are accessible
	if metrics.MessagesBefore != 100 {
		t.Errorf("MessagesBefore = %d, want 100", metrics.MessagesBefore)
	}
	if metrics.MessagesSummarized != 80 {
		t.Errorf("MessagesSummarized = %d, want 80", metrics.MessagesSummarized)
	}
	if metrics.MessagesKept != 10 {
		t.Errorf("MessagesKept = %d, want 10", metrics.MessagesKept)
	}
	if metrics.TokensBefore != 5000 {
		t.Errorf("TokensBefore = %d, want 5000", metrics.TokensBefore)
	}
	if metrics.TokensAfter != 500 {
		t.Errorf("TokensAfter = %d, want 500", metrics.TokensAfter)
	}
	if metrics.CompressionRatio != 0.1 {
		t.Errorf("CompressionRatio = %v, want 0.1", metrics.CompressionRatio)
	}
	if metrics.DedupGroupsRemoved != 5 {
		t.Errorf("DedupGroupsRemoved = %d, want 5", metrics.DedupGroupsRemoved)
	}
	if len(metrics.RulesApplied) != 3 {
		t.Errorf("RulesApplied count = %d, want 3", len(metrics.RulesApplied))
	}
	if metrics.DurationMs != 1500 {
		t.Errorf("DurationMs = %d, want 1500", metrics.DurationMs)
	}
}
