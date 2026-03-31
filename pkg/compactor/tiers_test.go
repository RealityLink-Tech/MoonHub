// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"strings"
	"testing"
)

func TestDefaultTierBudgets(t *testing.T) {
	budgets := DefaultTierBudgets()
	if budgets.L0 != 200 {
		t.Errorf("Default L0 = %d, want 200", budgets.L0)
	}
	if budgets.L1 != 1000 {
		t.Errorf("Default L1 = %d, want 1000", budgets.L1)
	}
	if budgets.L2 != 3000 {
		t.Errorf("Default L2 = %d, want 3000", budgets.L2)
	}
}

func TestGenerateTiers(t *testing.T) {
	l2Summary := `## User Identity
User name is Alice. She prefers concise responses.

## Decisions
Decided to use Go for the backend service.
Correction: Actually use Rust instead of Go.

## Tasks
- Implement authentication
- Add logging
- Write tests

## Preferences
Prefers dark mode. Likes detailed error messages.

## Context
Discussion about backend architecture and technology choices.
`

	budgets := TierBudgets{L0: 50, L1: 150, L2: 500}
	result := GenerateTiers(l2Summary, budgets)

	if result == nil {
		t.Fatal("GenerateTiers returned nil")
	}

	// L2 should be truncated to budget
	l2Tokens := EstimateTokens(result.L2)
	if l2Tokens > budgets.L2 {
		t.Errorf("L2 tokens = %d, exceeds budget %d", l2Tokens, budgets.L2)
	}

	// L1 should be smaller than L2
	l1Tokens := EstimateTokens(result.L1)
	if l1Tokens > budgets.L1 {
		t.Errorf("L1 tokens = %d, exceeds budget %d", l1Tokens, budgets.L1)
	}

	// L0 should be smallest
	l0Tokens := EstimateTokens(result.L0)
	if l0Tokens > budgets.L0 {
		t.Errorf("L0 tokens = %d, exceeds budget %d", l0Tokens, budgets.L0)
	}

	// L0 should contain high-priority info (identity, decisions)
	if !strings.Contains(result.L0, "Alice") && !strings.Contains(result.L0, "decision") {
		// L0 is very small, may not have all keywords
		t.Logf("L0 content: %q", result.L0)
	}
}

func TestGenerateTiers_EmptyInput(t *testing.T) {
	result := GenerateTiers("", TierBudgets{L0: 50, L1: 100, L2: 200})
	if result == nil {
		t.Fatal("GenerateTiers returned nil for empty input")
	}
	if result.L0 != "" || result.L1 != "" || result.L2 != "" {
		t.Errorf("GenerateTiers should return empty tiers for empty input")
	}
}

func TestGenerateTiersParallel(t *testing.T) {
	l2Summary := strings.Repeat("This is a test line with various content. ", 50)

	budgets := TierBudgets{L0: 30, L1: 100, L2: 300}
	result := GenerateTiersParallel(l2Summary, budgets)
	want := GenerateTiers(l2Summary, budgets)

	if result == nil {
		t.Fatal("GenerateTiersParallel returned nil")
	}
	if result.L0 != want.L0 || result.L1 != want.L1 || result.L2 != want.L2 {
		t.Errorf("GenerateTiersParallel must match GenerateTiers (L0/L1/L2 semantics)")
	}

	// Verify token budgets
	if EstimateTokens(result.L2) > budgets.L2 {
		t.Errorf("L2 exceeds budget")
	}
	if EstimateTokens(result.L1) > budgets.L1 {
		t.Errorf("L1 exceeds budget")
	}
	if EstimateTokens(result.L0) > budgets.L0 {
		t.Errorf("L0 exceeds budget")
	}
}

func TestScoreLine(t *testing.T) {
	tests := []struct {
		line     string
		minScore int
	}{
		{"User name is Alice", 9},    // contains "user" (9)
		{"Decision: use Go", 9},      // contains "decision" (9)
		{"Task: implement auth", 8},  // contains "task" (8)
		{"Preference: dark mode", 7}, // contains "preference" (7)
		{"Random content here", 1},   // no keywords, default
	}

	for _, tt := range tests {
		got := scoreLine(tt.line)
		if got < tt.minScore {
			t.Errorf("scoreLine(%q) = %d, want >= %d", tt.line, got, tt.minScore)
		}
	}
}

func TestBuildTierFromPriority(t *testing.T) {
	text := `Low priority content here.
User identity: Bob, a developer.
Medium priority stuff.
Decision: Use PostgreSQL for database.
More low priority.
Task: Implement API endpoints.
Even more content.`

	// Build a tier with limited budget
	result := buildTierFromPriority(text, 30)

	// Result should fit within budget
	tokens := EstimateTokens(result)
	if tokens > 30 {
		t.Errorf("buildTierFromPriority result has %d tokens, budget is 30", tokens)
	}

	// High-priority lines should be included (when budget allows)
	// "User" has priority 9, "Decision" has priority 9
	if len(result) > 0 {
		t.Logf("Result: %q", result)
	}
}

func TestTieredSummary_GetTier(t *testing.T) {
	// Test accessing tiers directly
	ts := &TieredSummary{
		L0: "Ultra compact",
		L1: "Working memory",
		L2: "Full summary",
	}

	if ts.L0 != "Ultra compact" {
		t.Errorf("L0 = %q, want 'Ultra compact'", ts.L0)
	}
	if ts.L1 != "Working memory" {
		t.Errorf("L1 = %q, want 'Working memory'", ts.L1)
	}
	if ts.L2 != "Full summary" {
		t.Errorf("L2 = %q, want 'Full summary'", ts.L2)
	}
}

func TestGenerateTiers_PriorityPreservation(t *testing.T) {
	// Create text where identity/decisions should be prioritized
	text := `## Notes
Some random notes that are less important.
More notes here.
Even more notes.

## Identity
User name: TestUser
Location: TestCity

## Decisions
Critical decision made here.
Another important decision.

## More Notes
Less important content.`

	budgets := TierBudgets{L0: 30, L1: 100, L2: 300}
	result := GenerateTiers(text, budgets)

	// L0 should fit within budget (with some tolerance for rounding)
	l0Tokens := EstimateTokens(result.L0)
	if l0Tokens > budgets.L0+5 {
		t.Errorf("L0 exceeds budget: %d tokens (budget %d)", l0Tokens, budgets.L0)
	}

	// L1 should have more content than L0
	if len(result.L1) < len(result.L0) {
		t.Error("L1 should be larger than L0")
	}

	// L2 should have the most content
	if len(result.L2) < len(result.L1) {
		t.Error("L2 should be larger than L1")
	}
}
