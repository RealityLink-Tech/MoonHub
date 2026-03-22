// MoonHub - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 MoonHub contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/adaptive_memory"
	"github.com/RealityLink-Tech/MoonHub/pkg/fileutil"
	"github.com/RealityLink-Tech/MoonHub/pkg/learning"
	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
// - Adaptive memory: memory/adaptive.db (SQLite with FTS5)
// - Learning engine: memory/learning.db (SQLite with FTS5)
type MemoryStore struct {
	workspace      string
	memoryDir      string
	memoryFile     string
	adaptiveMemory *adaptive_memory.Engine
	adaptiveConfig *adaptive_memory.Config
	learningEngine *learning.Engine
	learningConfig *learning.Config
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0o755)

	store := &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
	}

	// Initialize adaptive memory
	store.initAdaptiveMemory()

	// Initialize learning engine
	store.initLearningEngine()

	return store
}

// initAdaptiveMemory initializes the adaptive memory engine
func (ms *MemoryStore) initAdaptiveMemory() {
	dbPath := filepath.Join(ms.memoryDir, "adaptive.db")

	config := adaptive_memory.DefaultConfig(dbPath)
	config.EnableChinese = true // Enable Chinese segmentation by default

	// Apply custom config if provided
	if ms.adaptiveConfig != nil {
		config = *ms.adaptiveConfig
	}

	engine, err := adaptive_memory.NewMemoryEngine(config)
	if err != nil {
		logger.WarnCF("agent", "Failed to initialize adaptive memory, falling back to legacy memory.",
			map[string]any{"error": err.Error()})
		return
	}

	ms.adaptiveMemory = engine

	// Run migration from MEMORY.md if needed
	migrator := adaptive_memory.NewMigrator(engine, ms.memoryFile)
	if err := migrator.Migrate(); err != nil {
		logger.WarnCF("agent", "Failed to migrate MEMORY.md to adaptive memory.",
			map[string]any{"error": err.Error()})
	}

	logger.DebugCF("agent", "Adaptive memory initialized",
		map[string]any{"db_path": dbPath, "chinese_enabled": config.EnableChinese})
}

// initLearningEngine initializes the self-improving learning engine
func (ms *MemoryStore) initLearningEngine() {
	dbPath := filepath.Join(ms.memoryDir, "learning.db")

	config := learning.DefaultConfig(dbPath)

	// Apply custom config if provided
	if ms.learningConfig != nil {
		config = *ms.learningConfig
	}

	engine, err := learning.NewEngine(config)
	if err != nil {
		logger.WarnCF("agent", "Failed to initialize learning engine.",
			map[string]any{"error": err.Error()})
		return
	}

	ms.learningEngine = engine

	logger.DebugCF("agent", "Learning engine initialized",
		map[string]any{"db_path": dbPath})
}

// SetAdaptiveConfig sets custom adaptive memory configuration
func (ms *MemoryStore) SetAdaptiveConfig(config *adaptive_memory.Config) {
	ms.adaptiveConfig = config
}

// GetAdaptiveMemory returns the adaptive memory engine
func (ms *MemoryStore) GetAdaptiveMemory() *adaptive_memory.Engine {
	return ms.adaptiveMemory
}

// HasAdaptiveMemory returns true if adaptive memory is available
func (ms *MemoryStore) HasAdaptiveMemory() bool {
	return ms.adaptiveMemory != nil
}

// SetLearningConfig sets custom learning engine configuration
func (ms *MemoryStore) SetLearningConfig(config *learning.Config) {
	ms.learningConfig = config
}

// GetLearningEngine returns the learning engine
func (ms *MemoryStore) GetLearningEngine() *learning.Engine {
	return ms.learningEngine
}

// HasLearningEngine returns true if learning engine is available
func (ms *MemoryStore) HasLearningEngine() bool {
	return ms.learningEngine != nil
}

// RecordMemoryEvent records a new memory event to the adaptive memory system
func (ms *MemoryStore) RecordMemoryEvent(eventType adaptive_memory.EpisodicEventType, content string, outcome *string, importance *float64) (string, error) {
	if ms.adaptiveMemory == nil {
		return "", fmt.Errorf("adaptive memory not initialized")
	}

	return ms.adaptiveMemory.RecordEvent("default", adaptive_memory.EventInput{
		Type:       eventType,
		Content:    content,
		Outcome:    outcome,
		Importance: importance,
	})
}

// SearchMemory searches the adaptive memory for relevant memories
func (ms *MemoryStore) SearchMemory(query string, limit int) ([]adaptive_memory.MemorySearchResult, error) {
	if ms.adaptiveMemory == nil {
		return nil, fmt.Errorf("adaptive memory not initialized")
	}

	return ms.adaptiveMemory.Search("default", query, limit)
}

// ConsolidateMemory performs memory consolidation (decay, prune, merge)
func (ms *MemoryStore) ConsolidateMemory() (adaptive_memory.ConsolidationResult, error) {
	if ms.adaptiveMemory == nil {
		return adaptive_memory.ConsolidationResult{}, fmt.Errorf("adaptive memory not initialized")
	}

	return ms.adaptiveMemory.Consolidate("default")
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *MemoryStore) getTodayFile() string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	filePath := filepath.Join(ms.memoryDir, monthDir, today+".md")
	return filePath
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadLongTerm() string {
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(content string) error {
	// Use unified atomic write utility with explicit sync for flash storage reliability.
	// Using 0o600 (owner read/write only) for secure default permissions.
	return fileutil.WriteFileAtomic(ms.memoryFile, []byte(content), 0o600)
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadToday() string {
	todayFile := ms.getTodayFile()
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return err
	}

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(todayFile, []byte(newContent), 0o600)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	var sb strings.Builder
	first := true

	for i := range days {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			if !first {
				sb.WriteString("\n\n---\n\n")
			}
			sb.Write(data)
			first = false
		}
	}

	return sb.String()
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes adaptive memory (if available) or falls back to long-term memory and recent daily notes.
func (ms *MemoryStore) GetMemoryContext() string {
	// Try adaptive memory first
	if ms.adaptiveMemory != nil {
		ctx, err := ms.adaptiveMemory.GetContextForAgent("default", nil)
		if err == nil && ctx != "" {
			return ctx
		}
	}

	// Fallback to legacy memory
	longTerm := ms.ReadLongTerm()
	recentNotes := ms.GetRecentDailyNotes(3)

	if longTerm == "" && recentNotes == "" {
		return ""
	}

	var sb strings.Builder

	if longTerm != "" {
		sb.WriteString("## Long-term Memory\n\n")
		sb.WriteString(longTerm)
	}

	if recentNotes != "" {
		if longTerm != "" {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## Recent Daily Notes\n\n")
		sb.WriteString(recentNotes)
	}

	return sb.String()
}

// GetMemoryContextWithQuery returns formatted memory context with query-based retrieval
func (ms *MemoryStore) GetMemoryContextWithQuery(query string) string {
	// Try adaptive memory with query
	if ms.adaptiveMemory != nil && query != "" {
		ctx, err := ms.adaptiveMemory.GetContextForAgent("default", &query)
		if err == nil && ctx != "" {
			return ctx
		}
	}

	// Fallback to regular context
	return ms.GetMemoryContext()
}

// Close closes the memory store and releases resources
func (ms *MemoryStore) Close() error {
	var errs []error

	if ms.learningEngine != nil {
		if err := ms.learningEngine.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if ms.adaptiveMemory != nil {
		if err := ms.adaptiveMemory.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing memory store: %v", errs)
	}
	return nil
}
