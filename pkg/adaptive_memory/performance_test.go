// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Performance and Memory Tests
// License: MIT

package adaptive_memory

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestMemoryUsage_EngineStartup tests memory usage during engine startup
func TestMemoryUsage_EngineStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	before := float64(m.Alloc) / 1024 / 1024
	t.Logf("Before engine creation: Alloc=%.2fMB", before)

	tmpDir, err := os.MkdirTemp("", "memory_perf_test")
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

	runtime.ReadMemStats(&m)
	after := float64(m.Alloc) / 1024 / 1024
	delta := after - before
	t.Logf("After engine (no Chinese): Alloc=%.2fMB, Delta=%.2fMB", after, delta)

	if delta > 5 {
		t.Logf("Warning: Engine creation used more than 5MB without Chinese support")
	}
}

// TestMemoryUsage_WithChinese tests memory usage with Chinese segmentation enabled
func TestMemoryUsage_WithChinese(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	before := float64(m.Alloc) / 1024 / 1024
	t.Logf("Before engine with Chinese: Alloc=%.2fMB", before)

	tmpDir, err := os.MkdirTemp("", "memory_perf_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	config := DefaultConfig(dbPath)
	config.EnableChinese = true

	engine, err := NewMemoryEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	runtime.ReadMemStats(&m)
	after := float64(m.Alloc) / 1024 / 1024
	delta := after - before
	t.Logf("After engine with Chinese: Alloc=%.2fMB, Delta=%.2fMB", after, delta)

	// Note: GSE dictionary loads ~190MB on first load
	// This is a known behavior of go-ego/gse library
	// Consider using lazy loading or nogse build tag for constrained environments
	if delta > 200 {
		t.Logf("Warning: GSE dictionary uses %.2fMB memory. Consider using nogse build tag.", delta)
	} else if delta > 100 {
		t.Logf("Note: GSE dictionary loaded, using %.2fMB memory (expected ~190MB)", delta)
	}
}

// TestMemoryUsage_ManyRecords tests memory with many records
func TestMemoryUsage_ManyRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	before := float64(m.Alloc) / 1024 / 1024

	tmpDir, err := os.MkdirTemp("", "memory_perf_test")
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

	// Record 1000 events
	for i := 0; i < 1000; i++ {
		_, err := engine.RecordEvent("test-user", EventInput{
			Type:    EventTypeFactStored,
			Content: fmt.Sprintf("Test record number %d with some content to test memory usage", i),
		})
		if err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
	}

	runtime.ReadMemStats(&m)
	after := float64(m.Alloc) / 1024 / 1024
	delta := after - before
	t.Logf("After 1000 records: Alloc=%.2fMB, Delta=%.2fMB", after, delta)
	t.Logf("Average per record: %.2fKB", float64(delta*1024)/1000)
}

// TestPerformance_RecordingLatency tests recording latency
func TestPerformance_RecordingLatency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latency_test")
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

	iterations := 100
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := engine.RecordEvent("test-user", EventInput{
			Type:    EventTypeFactStored,
			Content: fmt.Sprintf("Latency test record %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
		totalDuration += time.Since(start)
	}

	avgLatency := totalDuration / time.Duration(iterations)
	t.Logf("Average recording latency: %v", avgLatency)
	t.Logf("Total for %d records: %v", iterations, totalDuration)

	if avgLatency > time.Millisecond {
		t.Errorf("Recording latency too high: %v (expected < 1ms)", avgLatency)
	}
}

// TestPerformance_SearchLatency tests search latency
func TestPerformance_SearchLatency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "latency_test")
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

	// Pre-populate with 500 records
	for i := 0; i < 500; i++ {
		_, err := engine.RecordEvent("test-user", EventInput{
			Type:    EventTypeFactStored,
			Content: fmt.Sprintf("Search test record about programming and development %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
	}

	iterations := 50
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := engine.Search("test-user", "programming", 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		totalDuration += time.Since(start)
	}

	avgLatency := totalDuration / time.Duration(iterations)
	t.Logf("Average search latency: %v", avgLatency)

	// CI / shared runners vary (disk, CPU); 8ms avg leaves headroom over typical ~3–6ms locally
	// while still catching serious regressions on 500 rows × FTS + hybrid scoring.
	if avgLatency > 8*time.Millisecond {
		t.Errorf("Search latency too high: %v (expected < 8ms)", avgLatency)
	}
}

// TestPerformance_Consolidation tests consolidation performance
func TestPerformance_Consolidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "consolidation_test")
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

	// Pre-populate with 100 records
	for i := 0; i < 100; i++ {
		_, err := engine.RecordEvent("test-user", EventInput{
			Type:    EventTypeFactStored,
			Content: fmt.Sprintf("Consolidation test record %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}
	}

	start := time.Now()
	result, err := engine.Consolidate("test-user")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Consolidation failed: %v", err)
	}

	t.Logf("Consolidation completed in %v", duration)
	t.Logf("Results: Decayed=%d, Pruned=%d, Merged=%d",
		result.Decayed, result.Pruned, result.Merged)

	if duration > 500*time.Millisecond {
		t.Errorf("Consolidation too slow: %v (expected < 500ms)", duration)
	}
}
