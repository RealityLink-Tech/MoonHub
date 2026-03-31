// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Unit Tests
// License: MIT

package adaptive_memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryEngine_BasicOperations(t *testing.T) {
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "adaptive_memory_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	config := DefaultConfig(dbPath)
	config.EnableChinese = false // Disable Chinese for simpler testing

	engine, err := NewMemoryEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test RecordEvent - use high importance event type for GetContextForAgent test
	id, err := engine.RecordEvent("test-user", EventInput{
		Type:    EventTypeCorrection, // Importance 0.9, needed for GetContextForAgent
		Content: "This is a test fact",
	})
	if err != nil {
		t.Fatalf("Failed to record event: %v", err)
	}
	if id == "" {
		t.Error("Expected non-empty ID")
	}

	// Test GetEvent
	record, err := engine.GetEvent(id)
	if err != nil {
		t.Fatalf("Failed to get event: %v", err)
	}
	if record == nil {
		t.Fatal("Expected non-nil record")
	}
	if record.Content != "This is a test fact" {
		t.Errorf("Expected content 'This is a test fact', got '%s'", record.Content)
	}
	if record.Importance != 0.9 {
		t.Errorf("Expected importance 0.9, got %f", record.Importance)
	}

	// Test Search
	results, err := engine.Search("test-user", "test fact", 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected at least one search result")
	}

	// Test GetContextForAgent
	ctx, err := engine.GetContextForAgent("test-user", nil)
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}
	if ctx == "" {
		t.Error("Expected non-empty context")
	}
}

func TestMemoryEngine_EventTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "adaptive_memory_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	config := DefaultConfig(dbPath)
	config.EnableChinese = false

	engine, err := NewMemoryEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	tests := []struct {
		eventType     EpisodicEventType
		expectedScore float64
	}{
		{EventTypeCorrection, 0.9},
		{EventTypePreferenceLearned, 0.8},
		{EventTypeFactStored, 0.6},
		{EventTypeTaskCompleted, 0.5},
		{EventTypeDelegationResult, 0.5},
	}

	for _, tc := range tests {
		id, err := engine.RecordEvent("test-user", EventInput{
			Type:    tc.eventType,
			Content: "Test content for " + string(tc.eventType),
		})
		if err != nil {
			t.Fatalf("Failed to record %s event: %v", tc.eventType, err)
		}

		record, err := engine.GetEvent(id)
		if err != nil {
			t.Fatalf("Failed to get %s event: %v", tc.eventType, err)
		}

		if record.Importance != tc.expectedScore {
			t.Errorf("Event %s: expected importance %f, got %f",
				tc.eventType, tc.expectedScore, record.Importance)
		}
	}
}

func TestScoring_TemporalDecay(t *testing.T) {
	now := NowMs()

	// Test 1: Fresh memory should have high score
	freshScore := ComputeTemporalScore(now, 0, now)
	if freshScore < 0.95 {
		t.Errorf("Fresh memory score too low: %f", freshScore)
	}

	// Test 2: Old memory should have lower score
	oldTime := now - int64(7*MsPerDay) // 7 days old
	oldScore := ComputeTemporalScore(oldTime, 0, now)
	if oldScore > 0.75 {
		t.Errorf("Old memory score too high: %f (expected < 0.75)", oldScore)
	}

	// Test 3: Frequent access should boost score
	frequentScore := ComputeTemporalScore(now-7*MsPerDay, 10, now)
	if frequentScore <= oldScore {
		t.Errorf("Frequent access should boost score: %f vs %f", frequentScore, oldScore)
	}

	// Test 4: Score should be clamped to 1.0
	highScore := ComputeTemporalScore(now, 100, now)
	if highScore > 1.0 {
		t.Errorf("Score should be clamped to 1.0, got %f", highScore)
	}
}

func TestScoring_ContentSimilarity(t *testing.T) {
	tests := []struct {
		a, b     string
		expected float64
	}{
		{"hello world", "hello world", 1.0},
		{"hello world", "hello", 0.5},          // intersection=1 ("hello"), union=2, 1/2=0.5
		{"hello world", "goodbye world", 0.33}, // intersection=1 ("world"), union=3, 1/3≈0.33
		{"hello", "goodbye", 0.0},
		{"", "", 0.0},
	}

	for _, tc := range tests {
		similarity := ContentSimilarity(tc.a, tc.b)
		// Allow some tolerance for floating point comparison
		if similarity < tc.expected-0.1 || similarity > tc.expected+0.1 {
			t.Errorf("ContentSimilarity(%q, %q) = %f, expected ~%f",
				tc.a, tc.b, similarity, tc.expected)
		}
	}
}

func TestChineseSegmenter_ContainsChinese(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"Hello World", false},
		{"你好世界", true},
		{"Hello 世界", true},
		{"123456", false},
		{"", false},
	}

	for _, tc := range tests {
		result := ContainsChinese(tc.text)
		if result != tc.expected {
			t.Errorf("ContainsChinese(%q) = %v, expected %v", tc.text, result, tc.expected)
		}
	}
}

func TestDefaultImportance(t *testing.T) {
	tests := []struct {
		eventType EpisodicEventType
		expected  float64
	}{
		{EventTypeCorrection, 0.9},
		{EventTypePreferenceLearned, 0.8},
		{EventTypeFactStored, 0.6},
		{EventTypeTaskCompleted, 0.5},
		{EventTypeDelegationResult, 0.5},
		// New event types
		{EventTypeUserFeedback, 0.85},
		{EventTypeInsight, 0.7},
		{EventTypeReminder, 0.6},
		{EventTypeErrorLearned, 0.75},
		{EventTypeContextUpdate, 0.5},
		{"unknown_type", 0.5},
	}

	for _, tc := range tests {
		result := GetDefaultImportance(tc.eventType)
		if result != tc.expected {
			t.Errorf("GetDefaultImportance(%s) = %f, expected %f",
				tc.eventType, result, tc.expected)
		}
	}
}

func TestMemoryEngine_NewEventTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "adaptive_memory_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	config := DefaultConfig(dbPath)
	config.EnableChinese = false

	engine, err := NewMemoryEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test user_feedback event
	id, err := engine.RecordEvent("test-user", EventInput{
		Type:    EventTypeUserFeedback,
		Content: "User said the previous answer was helpful",
	})
	if err != nil {
		t.Fatalf("Failed to record user_feedback event: %v", err)
	}
	record, err := engine.GetEvent(id)
	if err != nil {
		t.Fatalf("Failed to get user_feedback event: %v", err)
	}
	if record.Importance != 0.85 {
		t.Errorf("Expected importance 0.85 for user_feedback, got %f", record.Importance)
	}

	// Test insight event
	id, err = engine.RecordEvent("test-user", EventInput{
		Type:    EventTypeInsight,
		Content: "Noticed user prefers Python over JavaScript",
	})
	if err != nil {
		t.Fatalf("Failed to record insight event: %v", err)
	}
	record, err = engine.GetEvent(id)
	if err != nil {
		t.Fatalf("Failed to get insight event: %v", err)
	}
	if record.Importance != 0.7 {
		t.Errorf("Expected importance 0.7 for insight, got %f", record.Importance)
	}

	// Test error_learned event
	id, err = engine.RecordEvent("test-user", EventInput{
		Type:    EventTypeErrorLearned,
		Content: "User corrected: don't use verbose logging",
	})
	if err != nil {
		t.Fatalf("Failed to record error_learned event: %v", err)
	}
	record, err = engine.GetEvent(id)
	if err != nil {
		t.Fatalf("Failed to get error_learned event: %v", err)
	}
	if record.Importance != 0.75 {
		t.Errorf("Expected importance 0.75 for error_learned, got %f", record.Importance)
	}

	// Test reminder event
	id, err = engine.RecordEvent("test-user", EventInput{
		Type:    EventTypeReminder,
		Content: "Scheduled reminder: weekly report due on Friday",
	})
	if err != nil {
		t.Fatalf("Failed to record reminder event: %v", err)
	}
	record, err = engine.GetEvent(id)
	if err != nil {
		t.Fatalf("Failed to get reminder event: %v", err)
	}
	if record.Importance != 0.6 {
		t.Errorf("Expected importance 0.6 for reminder, got %f", record.Importance)
	}

	// Test context_update event
	id, err = engine.RecordEvent("test-user", EventInput{
		Type:    EventTypeContextUpdate,
		Content: "Context updated: project switched to new-api",
	})
	if err != nil {
		t.Fatalf("Failed to record context_update event: %v", err)
	}
	record, err = engine.GetEvent(id)
	if err != nil {
		t.Fatalf("Failed to get context_update event: %v", err)
	}
	if record.Importance != 0.5 {
		t.Errorf("Expected importance 0.5 for context_update, got %f", record.Importance)
	}
}

func TestSearchStoreLimit(t *testing.T) {
	tests := []struct {
		requested, ftsCap, want int
	}{
		{10, 50, 40},  // 4*10 >= 10+16 → 40
		{5, 50, 21},   // 4*5 < 5+16 → 21
		{50, 50, 50},  // capped at ftsCap
		{100, 50, 50}, // cannot fetch more than ftsCap
		{1, 50, 17},   // 4 < 1+16 → 17
	}
	for _, tt := range tests {
		if got := searchStoreLimit(tt.requested, tt.ftsCap); got != tt.want {
			t.Errorf("searchStoreLimit(%d,%d) = %d, want %d", tt.requested, tt.ftsCap, got, tt.want)
		}
	}
}
