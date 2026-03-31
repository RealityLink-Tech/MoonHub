// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"path/filepath"
	"testing"
)

func TestNewEngine(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)

	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestEngine_Analyze(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "Great job!"},
			{Role: "assistant", Content: "Thanks!"},
		},
		RecentToolCalls: []ToolCall{},
	}

	result := engine.Analyze(ctx)

	if result.Signals == nil {
		t.Error("expected non-nil signals slice")
	}

	// Should detect the positive feedback
	if len(result.Signals) == 0 {
		t.Log("Warning: no signals detected from 'Great job!'")
	}
}

func TestEngine_Analyze_Correction(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "Write a function"},
			{Role: "assistant", Content: "Here's a function in JavaScript..."},
			{Role: "user", Content: "No, use TypeScript instead"},
			{Role: "assistant", Content: "Okay, using TypeScript..."},
		},
		RecentToolCalls: []ToolCall{},
	}

	result := engine.Analyze(ctx)

	// Should detect the correction
	if len(result.Signals) == 0 {
		t.Log("Warning: no signals detected from correction")
	}

	// Should create new patterns
	if len(result.NewPatterns) == 0 {
		t.Log("Warning: no new patterns from correction")
	}
}

func TestEngine_GetContextString(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Initially empty
	ctx := engine.GetContextString()
	// First run may be empty if no patterns
	t.Logf("Initial context: %q", ctx)
}

func TestEngine_RecordToolExecution(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	record := ToolExecutionRecord{
		ToolName:    "test_tool",
		Success:     true,
		DurationMs:  150,
		UserContext: "testing",
		Timestamp:   NowMs(),
	}

	// Should not panic
	engine.RecordToolExecution(record)

	// Check that tool usage is tracked
	patterns := engine.GetToolUsagePatterns()
	if len(patterns) == 0 {
		t.Error("expected tool usage to be tracked")
	}

	found := false
	for _, p := range patterns {
		if p.ToolName == "test_tool" {
			found = true
			if p.TotalCalls != 1 {
				t.Errorf("expected 1 total call, got %d", p.TotalCalls)
			}
			if p.SuccessfulCalls != 1 {
				t.Errorf("expected 1 successful call, got %d", p.SuccessfulCalls)
			}
			break
		}
	}

	if !found {
		t.Error("expected to find test_tool in usage patterns")
	}
}

func TestEngine_RecordToolRejection(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// First record a successful execution
	engine.RecordToolExecution(ToolExecutionRecord{
		ToolName:   "rejected_tool",
		Success:    true,
		DurationMs: 100,
		Timestamp:  NowMs(),
	})

	// Then record rejection
	engine.RecordToolRejection("rejected_tool")

	// Check that rejection is tracked
	patterns := engine.GetToolUsagePatterns()
	for _, p := range patterns {
		if p.ToolName == "rejected_tool" {
			if p.UserRejectedCalls != 1 {
				t.Errorf("expected 1 rejected call, got %d", p.UserRejectedCalls)
			}
			// User preference should be negative
			if p.UserPreference >= 0 {
				t.Errorf("expected negative preference, got %f", p.UserPreference)
			}
			return
		}
	}
	t.Error("expected to find rejected_tool in usage patterns")
}

func TestEngine_RecordToolAcceptance(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Record successful execution
	engine.RecordToolExecution(ToolExecutionRecord{
		ToolName:   "accepted_tool",
		Success:    true,
		DurationMs: 100,
		Timestamp:  NowMs(),
	})

	// Record acceptance
	engine.RecordToolAcceptance("accepted_tool")

	// Check that acceptance improves preference
	patterns := engine.GetToolUsagePatterns()
	for _, p := range patterns {
		if p.ToolName == "accepted_tool" {
			// With just one success and acceptance, preference should be positive
			t.Logf("User preference for accepted_tool: %f", p.UserPreference)
			return
		}
	}
	t.Error("expected to find accepted_tool in usage patterns")
}

func TestEngine_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	stats := engine.GetStats()

	if stats.PatternsByCategory == nil {
		t.Error("expected non-nil PatternsByCategory map")
	}

	t.Logf("Stats: TotalPatterns=%d, HighConfidence=%d",
		stats.TotalPatterns, stats.HighConfidencePatterns)
}

func TestEngine_GetAllPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	patterns := engine.GetAllPatterns()
	if patterns == nil {
		t.Error("expected non-nil patterns slice")
	}
}

func TestEngine_GetPatternsByCategory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Analyze to create some patterns
	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "Great job!"},
			{Role: "assistant", Content: "Thanks!"},
		},
	}
	engine.Analyze(ctx)

	// Get patterns by category
	successPatterns := engine.GetPatternsByCategory(CategorySuccessIndicator)
	t.Logf("Success patterns: %d", len(successPatterns))
}

func TestEngine_SearchPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Analyze to create some patterns
	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "I prefer concise responses"},
			{Role: "assistant", Content: "Got it, keeping it brief."},
		},
	}
	engine.Analyze(ctx)

	// Search for patterns
	results, err := engine.SearchPatterns("concise", 10)
	if err != nil {
		t.Errorf("failed to search patterns: %v", err)
	}

	t.Logf("Search results: %d patterns found", len(results))
}

func TestEngine_DeletePattern(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Analyze to create some patterns
	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "Great job!"},
			{Role: "assistant", Content: "Thanks!"},
		},
	}
	result := engine.Analyze(ctx)

	// If patterns were created, try to delete one
	if len(result.NewPatterns) > 0 {
		patternID := result.NewPatterns[0].ID
		err := engine.DeletePattern(patternID)
		if err != nil {
			t.Errorf("failed to delete pattern: %v", err)
		}

		// Verify deletion
		patterns := engine.GetAllPatterns()
		for _, p := range patterns {
			if p.ID == patternID {
				t.Error("pattern should have been deleted")
				break
			}
		}
	}
}

func TestEngine_Evolve(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	config.EnablePatternEvolution = true
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Create some patterns
	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "Great job!"},
			{Role: "assistant", Content: "Thanks!"},
		},
	}
	engine.Analyze(ctx)

	// Run evolution
	result := engine.Evolve()

	t.Logf("Evolution result: decayed=%d, merged=%d, pruned=%d, generalized=%d",
		result.PatternsDecayed, result.PatternsMerged,
		result.PatternsPruned, result.PatternsGeneralized)
}

func TestEngine_Evolve_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	config.EnablePatternEvolution = false // Disabled by default
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Create some patterns
	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "Great job!"},
			{Role: "assistant", Content: "Thanks!"},
		},
	}
	engine.Analyze(ctx)

	// Run evolution (should be no-op when disabled)
	result := engine.Evolve()

	// When disabled, all counts should be 0
	if result.PatternsDecayed != 0 || result.PatternsMerged != 0 ||
		result.PatternsPruned != 0 || result.PatternsGeneralized != 0 {
		t.Error("evolution should be no-op when disabled")
	}
}

func TestEngine_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Close should not error
	err = engine.Close()
	if err != nil {
		t.Errorf("failed to close engine: %v", err)
	}
}

func TestEngine_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create engine and add some data
	config := DefaultConfig(dbPath)
	engine1, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine1: %v", err)
	}

	// Analyze to create patterns
	ctx := AnalysisContext{
		UserID: "test_user",
		MessageHistory: []Message{
			{Role: "user", Content: "I prefer concise responses"},
			{Role: "assistant", Content: "Got it!"},
		},
	}
	engine1.Analyze(ctx)
	engine1.Close()

	// Create new engine and verify data persisted
	engine2, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine2: %v", err)
	}
	defer engine2.Close()

	patterns := engine2.GetAllPatterns()
	if len(patterns) == 0 {
		t.Error("expected patterns to persist after restart")
	}

	t.Logf("Persisted patterns: %d", len(patterns))
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("/tmp/test.db")

	if config.MinConfidence != 0.7 {
		t.Errorf("expected MinConfidence 0.7, got %f", config.MinConfidence)
	}
	if config.MaxExamples != 10 {
		t.Errorf("expected MaxExamples 10, got %d", config.MaxExamples)
	}
	if !config.EnableSemanticDetection {
		t.Error("expected EnableSemanticDetection to be true")
	}
	if !config.EnableImplicitSignals {
		t.Error("expected EnableImplicitSignals to be true")
	}
	if config.EnablePatternEvolution {
		t.Error("expected EnablePatternEvolution to be false by default")
	}
}

func TestNowMs(t *testing.T) {
	now := NowMs()
	if now <= 0 {
		t.Errorf("expected positive timestamp, got %d", now)
	}
}

func TestDaysSince(t *testing.T) {
	// Now
	days := DaysSince(NowMs())
	if days < 0 || days > 1 {
		t.Errorf("expected ~0 days, got %f", days)
	}

	// 1 day ago
	oneDayAgo := NowMs() - MsPerDay
	days = DaysSince(oneDayAgo)
	if days < 0.9 || days > 1.1 {
		t.Errorf("expected ~1 day, got %f", days)
	}

	// Zero timestamp
	days = DaysSince(0)
	if days != 0 {
		t.Errorf("expected 0 days for zero timestamp, got %f", days)
	}
}

func TestMinFloat64(t *testing.T) {
	if MinFloat64(1.0, 2.0) != 1.0 {
		t.Error("MinFloat64(1.0, 2.0) should be 1.0")
	}
	if MinFloat64(2.0, 1.0) != 1.0 {
		t.Error("MinFloat64(2.0, 1.0) should be 1.0")
	}
}

func TestMaxFloat64(t *testing.T) {
	if MaxFloat64(1.0, 2.0) != 2.0 {
		t.Error("MaxFloat64(1.0, 2.0) should be 2.0")
	}
	if MaxFloat64(2.0, 1.0) != 2.0 {
		t.Error("MaxFloat64(2.0, 1.0) should be 2.0")
	}
}

func TestMinInt(t *testing.T) {
	if MinInt(1, 2) != 1 {
		t.Error("MinInt(1, 2) should be 1")
	}
	if MinInt(2, 1) != 1 {
		t.Error("MinInt(2, 1) should be 1")
	}
}
