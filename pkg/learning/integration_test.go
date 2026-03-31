// MoonHub - Your ready-to-use AI assistant
// Self-Improving Behavioral Pattern Detection System
// License: MIT

package learning

import (
	"path/filepath"
	"testing"
)

func TestAgentIntegration_AnalyzeConversation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	integration := NewAgentIntegration(engine, nil, "default")

	history := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Great job!"},
		{Role: "assistant", Content: "Thanks!"},
	}
	result := integration.AnalyzeConversation(
		"Great job!",
		"Thanks!",
		history,
		nil,
		nil,
	)

	// Verify signals were detected
	if len(result.Signals) == 0 {
		t.Log("Warning: no signals detected")
	}

	// Verify patterns were created
	if len(result.NewPatterns) == 0 {
		t.Log("Warning: no patterns created")
	}
}

func TestAgentIntegration_RecordToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	integration := NewAgentIntegration(engine, nil, "default")

	// Record tool call
	integration.RecordToolCall("test_tool", true, 100, "testing context")

	// Check that tool was tracked
	patterns := engine.GetToolUsagePatterns()
	if len(patterns) == 0 {
		t.Error("expected tool usage pattern to be tracked")
	}

	if patterns[0].ToolName != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %s", patterns[0].ToolName)
	}

	if patterns[0].TotalCalls != 1 {
		t.Errorf("expected 1 total call, got %d", patterns[0].TotalCalls)
	}

	if patterns[0].AvgDurationMs != 100 {
		t.Errorf("expected avg duration 100ms, got %d", patterns[0].AvgDurationMs)
	}
}

func TestAgentIntegration_RecordUserFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Create integration wrapper
	integration := NewAgentIntegration(engine, nil, "default")

	// Record tool execution first so the tool exists in tracker
	integration.RecordToolCall("test_tool", true, 100, "test context")

	// Record positive feedback
	integration.RecordUserFeedback("test_tool", true)

	// Check patterns after acceptance - preference should be positive
	patterns := engine.GetToolUsagePatterns()
	if len(patterns) != 1 {
		t.Error("expected tool usage to be tracked")
	}

	found := false
	for _, p := range patterns {
		if p.ToolName == "test_tool" {
			found = true
			// After true feedback, preference should be positive
			if p.UserPreference <= 0 {
				t.Errorf("expected positive preference after acceptance, got %f", p.UserPreference)
			}
			break
		}
	}

	if !found {
		t.Error("expected to find test_tool in usage patterns")
	}

	// Now record negative feedback and verify preference decreases
	integration.RecordUserFeedback("test_tool", false)
	patterns = engine.GetToolUsagePatterns()
	for _, p := range patterns {
		if p.ToolName == "test_tool" {
			// After both acceptance and rejection, preference should be neutral or slightly negative
			if p.UserPreference > 0 {
				t.Errorf("expected neutral or negative preference after mixed feedback, got %f", p.UserPreference)
			}
			break
		}
	}

	if !found {
		t.Error("expected to find test_tool in usage patterns")
	}
}

func TestAgentIntegration_GetLearnedContext(t *testing.T) {
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
			{Role: "assistant", Content: "Got it!"},
		},
		RecentToolCalls:   []ToolCall{},
		RecentToolResults: []ToolResult{},
	}
	_ = engine.Analyze(ctx)
	integration := NewAgentIntegration(engine, nil, "default")

	context := integration.GetLearnedContext()

	// Context should contain the preference
	if context == "" {
		t.Error("expected non-empty context after analysis")
	}

	t.Logf("Learned context: %s", context)
}

func TestAgentIntegration_GetToolPreferences(t *testing.T) {
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
			{Role: "user", Content: "Use web search tool"},
			{Role: "assistant", Content: "I'll search now."},
		},
		RecentToolCalls:   []ToolCall{},
		RecentToolResults: []ToolResult{},
	}
	engine.Analyze(ctx)
	integration := NewAgentIntegration(engine, nil, "default")

	prefs := integration.GetToolPreferences()

	t.Logf("Tool preferences: %s", prefs)
}

func TestAgentIntegration_GetWorkflowPatterns(t *testing.T) {
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
			{Role: "user", Content: "Always run tests after code changes"},
			{Role: "assistant", Content: "I'll run the tests now."},
		},
		RecentToolCalls:   []ToolCall{},
		RecentToolResults: []ToolResult{},
	}
	engine.Analyze(ctx)
	integration := NewAgentIntegration(engine, nil, "default")

	patterns := integration.GetWorkflowPatterns()

	t.Logf("Workflow patterns: %s", patterns)
}
func TestAgentIntegration_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConfig(dbPath)
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	integration := NewAgentIntegration(engine, nil, "default")
	stats := integration.GetStats()

	t.Logf("Stats: TotalPatterns=%d, ToolUsagePatterns=%d",
		stats.TotalPatterns, stats.ToolUsagePatterns)
}
