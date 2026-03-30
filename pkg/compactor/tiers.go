// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"sort"
	"strings"
)

// TierBudgets holds token budgets for each tier
type TierBudgets struct {
	L0 int // Ultra-compact (default: 200 tokens)
	L1 int // Working memory (default: 1000 tokens)
	L2 int // Full context (default: 3000 tokens)
}

// DefaultTierBudgets returns the default token budgets for tiers
func DefaultTierBudgets() TierBudgets {
	return TierBudgets{
		L0: 200,
		L1: 1000,
		L2: 3000,
	}
}

// TieredSummary holds all three tier summaries
type TieredSummary struct {
	L0 string // Ultra-compact (200 tokens): identity, critical decisions
	L1 string // Working memory (1000 tokens): decisions, actions, preferences
	L2 string // Full context (3000 tokens): complete summary
}

// SectionPriority maps keywords to importance scores.
// Higher = kept first when building compressed tiers.
var SectionPriority = map[string]int{
	// Identity / user facts
	"name": 10, "identity": 10, "user": 9,
	// Decisions
	"decision": 9, "correction": 9,
	// Active tasks
	"task": 8, "todo": 8, "action": 8,
	// Preferences
	"preference": 7, "like": 6, "dislike": 6,
	// Context
	"topic": 5, "conversation": 4, "summary": 4, "note": 3,
}

// scoreLine scores a line or section based on keyword matches.
func scoreLine(line string) int {
	lower := strings.ToLower(line)
	maxScore := 1 // Default priority for unmatched lines

	for keyword, priority := range SectionPriority {
		if strings.Contains(lower, keyword) {
			if priority > maxScore {
				maxScore = priority
			}
		}
	}

	return maxScore
}

// scoredLine holds a line with its score and original index
type scoredLine struct {
	line          string
	score         int
	originalIndex int
}

// GenerateTiers generates tiered summaries from a full L2 summary.
// L2 is the full summary (truncated to budget if needed).
// L1 is derived from L2 by selecting highest-priority lines.
// L0 is derived from L1 by selecting only the most critical facts.
func GenerateTiers(l2Summary string, budgets TierBudgets) *TieredSummary {
	// L2 is the full summary, just ensure it fits the budget
	l2 := TruncateToTokenBudget(l2Summary, budgets.L2)

	// L1: Keep highest-priority lines within budget
	l1 := buildTierFromPriority(l2, budgets.L1)

	// L0: Keep only the most critical facts
	l0 := buildTierFromPriority(l1, budgets.L0)

	return &TieredSummary{
		L0: l0,
		L1: l1,
		L2: l2,
	}
}

// GenerateTiersParallel matches GenerateTiers (L0 derived from L1) so parallel pre-processing
// elsewhere does not change tier semantics.
func GenerateTiersParallel(l2Summary string, budgets TierBudgets) *TieredSummary {
	return GenerateTiers(l2Summary, budgets)
}

// buildTierFromPriority builds a tier by selecting highest-priority lines that fit the budget.
func buildTierFromPriority(text string, tokenBudget int) string {
	// Split into non-empty lines
	lines := strings.Split(text, "\n")
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return ""
	}

	// Score lines and capture original index
	scored := make([]scoredLine, len(nonEmptyLines))
	for i, line := range nonEmptyLines {
		scored[i] = scoredLine{
			line:          line,
			score:         scoreLine(line),
			originalIndex: i,
		}
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Greedily fill until budget is reached
	type selectedLine struct {
		line          string
		originalIndex int
	}
	var selected []selectedLine
	currentTokens := 0

	for _, sl := range scored {
		lineTokens := EstimateTokens(sl.line)
		if currentTokens+lineTokens <= tokenBudget {
			selected = append(selected, selectedLine{
				line:          sl.line,
				originalIndex: sl.originalIndex,
			})
			currentTokens += lineTokens
		}
	}

	// Re-sort by original order to maintain coherent reading
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].originalIndex < selected[j].originalIndex
	})

	// Build result
	resultLines := make([]string, len(selected))
	for i, s := range selected {
		resultLines[i] = s.line
	}

	return strings.Join(resultLines, "\n")
}

// SelectTier selects the appropriate tier based on context pressure.
// Returns 0 (L0), 1 (L1), or 2 (L2) based on history length.
func SelectTier(historyLen int) int {
	// Simple heuristic: more history = more compression
	if historyLen > 40 {
		return 0 // L0: ultra-compact
	} else if historyLen > 20 {
		return 1 // L1: working memory
	}
	return 2 // L2: full summary
}

// SelectTierByTokens selects the appropriate tier based on token budget.
func SelectTierByTokens(availableTokens int, budgets TierBudgets) int {
	if availableTokens < budgets.L0 {
		return -1 // Not enough tokens for any tier
	} else if availableTokens < budgets.L1 {
		return 0 // L0 only
	} else if availableTokens < budgets.L2 {
		return 1 // L1
	}
	return 2 // L2
}

// GetTier returns the summary for the specified tier (0=L0, 1=L1, 2=L2).
func (ts *TieredSummary) GetTier(tier int) string {
	switch tier {
	case 0:
		return ts.L0
	case 1:
		return ts.L1
	case 2:
		return ts.L2
	default:
		return ts.L2
	}
}

// TokenCounts returns the estimated token count for each tier.
func (ts *TieredSummary) TokenCounts() (l0, l1, l2 int) {
	return EstimateTokens(ts.L0), EstimateTokens(ts.L1), EstimateTokens(ts.L2)
}

// IsEmpty checks if all tiers are empty.
func (ts *TieredSummary) IsEmpty() bool {
	return ts.L0 == "" && ts.L1 == "" && ts.L2 == ""
}
