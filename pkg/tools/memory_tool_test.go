package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/adaptive_memory"
)

// MockMemoryStore implements MemoryToolProvider for testing
type MockMemoryStore struct {
	engine *adaptive_memory.Engine
}

func (m *MockMemoryStore) GetAdaptiveMemory() *adaptive_memory.Engine {
	return m.engine
}

func setupMemoryToolTest(t *testing.T) (*MemoryTool, *adaptive_memory.Engine, func()) {
	tmpDir, err := os.MkdirTemp("", "memory_tool_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	dbPath := filepath.Join(tmpDir, "test.db")
	config := adaptive_memory.DefaultConfig(dbPath)
	config.EnableChinese = false

	engine, err := adaptive_memory.NewMemoryEngine(config)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create engine: %v", err)
	}

	mockStore := &MockMemoryStore{engine: engine}
	tool := NewMemoryTool(mockStore)
	return tool, engine, cleanup
}

func TestMemoryTool_NameAndParams(t *testing.T) {
	tool, _, cleanup := setupMemoryToolTest(t)
	defer cleanup()

	if tool.Name() != "memory" {
		t.Errorf("Expected tool name 'memory', got %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Errorf("Expected non-empty description")
	}
	params := tool.Parameters()
	if params == nil {
		t.Fatalf("Expected non-nil parameters")
	}
}

func TestMemoryTool_Record(t *testing.T) {
	tool, engine, cleanup := setupMemoryToolTest(t)
	defer cleanup()
	defer engine.Close()

	ctx := context.Background()
	result := tool.Execute(ctx, map[string]any{
		"action":     "record",
		"event_type": "preference_learned",
		"content":    "User prefers Go over Python",
	})

	if result.IsError {
		t.Fatalf("Record failed: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Error("Expected Silent result for record action")
	}
	if result.ForLLM == "" {
		t.Error("Expected non-empty ForLLM for record success")
	}
}

func TestMemoryTool_Search(t *testing.T) {
	tool, engine, cleanup := setupMemoryToolTest(t)
	defer cleanup()
	defer engine.Close()

	ctx := context.Background()

	// Record first
	_ = tool.Execute(ctx, map[string]any{
		"action":     "record",
		"event_type": "fact_stored",
		"content":    "User works on Go projects",
	})

	// Search
	result := tool.Execute(ctx, map[string]any{
		"action": "search",
		"query":  "Go",
		"limit":  float64(10),
	})

	if result.IsError {
		t.Fatalf("Search failed: %s", result.ForLLM)
	}
	if result.ForLLM == "" {
		t.Error("Expected non-empty search result")
	}
}

func TestMemoryTool_List(t *testing.T) {
	tool, engine, cleanup := setupMemoryToolTest(t)
	defer cleanup()
	defer engine.Close()

	ctx := context.Background()

	// Record first
	_ = tool.Execute(ctx, map[string]any{
		"action":     "record",
		"event_type": "fact_stored",
		"content":    "Test memory for list",
	})

	// List
	result := tool.Execute(ctx, map[string]any{
		"action": "list",
		"limit":  float64(5),
	})

	if result.IsError {
		t.Fatalf("List failed: %s", result.ForLLM)
	}
	if result.ForLLM == "" {
		t.Error("Expected non-empty list result")
	}
}

func TestMemoryTool_Consolidate(t *testing.T) {
	tool, engine, cleanup := setupMemoryToolTest(t)
	defer cleanup()
	defer engine.Close()

	ctx := context.Background()
	result := tool.Execute(ctx, map[string]any{"action": "consolidate"})

	if result.IsError {
		t.Fatalf("Consolidate failed: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Error("Expected Silent result for consolidate action")
	}
}

func TestMemoryTool_ErrorCases(t *testing.T) {
	tool, engine, cleanup := setupMemoryToolTest(t)
	defer cleanup()
	defer engine.Close()

	ctx := context.Background()

	t.Run("missing_action", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]any{})
		if !result.IsError {
			t.Error("Expected error for missing action")
		}
	})

	t.Run("record_missing_event_type", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]any{
			"action":  "record",
			"content": "some content",
		})
		if !result.IsError {
			t.Error("Expected error for record without event_type")
		}
	})

	t.Run("record_missing_content", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]any{
			"action":     "record",
			"event_type": "fact_stored",
		})
		if !result.IsError {
			t.Error("Expected error for record without content")
		}
	})

	t.Run("search_missing_query", func(t *testing.T) {
		result := tool.Execute(ctx, map[string]any{
			"action": "search",
		})
		if !result.IsError {
			t.Error("Expected error for search without query")
		}
	})

	t.Run("nil_engine", func(t *testing.T) {
		nilStore := &MockMemoryStore{engine: nil}
		nilTool := NewMemoryTool(nilStore)
		result := nilTool.Execute(ctx, map[string]any{
			"action":     "record",
			"event_type": "fact_stored",
			"content":    "test",
		})
		if !result.IsError {
			t.Error("Expected error when adaptive memory is nil")
		}
	})
}
