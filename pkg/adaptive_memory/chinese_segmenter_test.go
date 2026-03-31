// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Chinese Segmentation Quality Tests
// License: MIT

package adaptive_memory

import (
	"os"
	"path/filepath"
	"testing"
)

// TestChineseSegmentationQuality tests the quality of Chinese segmentation
// This test requires manual verification of results
func TestChineseSegmentationQuality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Chinese segmentation quality test in short mode")
	}

	// Create temp database
	tmpDir, err := os.MkdirTemp("", "chinese_segment_test")
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

	// Test cases for Chinese segmentation quality
	testCases := []struct {
		name     string
		content  string
		query    string
		expected bool // whether query should find the content
	}{
		// Basic Chinese word segmentation
		{
			name:     "simple_preference",
			content:  "用户喜欢简洁的代码风格",
			query:    "代码风格",
			expected: true,
		},
		{
			name:     "programming_term",
			content:  "项目使用Go语言开发",
			query:    "Go语言",
			expected: true,
		},
		{
			name:     "mixed_content",
			content:  "使用React框架构建前端应用",
			query:    "React",
			expected: true,
		},
		{
			name:     "technical_term",
			content:  "数据库连接池配置优化",
			query:    "数据库",
			expected: true,
		},
		{
			name:     "sentence_search",
			content:  "每天早上八点开始工作",
			query:    "早上",
			expected: true,
		},
		// Edge cases
		{
			name:     "partial_word",
			content:  "人工智能技术发展迅速",
			query:    "人工",
			expected: true,
		},
		{
			name:     "compound_word",
			content:  "深度学习模型训练",
			query:    "深度学习",
			expected: true,
		},
		{
			name:     "english_chinese_mix",
			content:  "使用API接口进行数据交互",
			query:    "API接口",
			expected: true,
		},
		// Should not match cases
		{
			name:     "no_match_different_topic",
			content:  "今天天气很好",
			query:    "编程",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Record the memory
			_, err := engine.RecordEvent("test-user", EventInput{
				Type:    EventTypeFactStored,
				Content: tc.content,
			})
			if err != nil {
				t.Fatalf("Failed to record event: %v", err)
			}

			// Search for the query
			results, err := engine.Search("test-user", tc.query, 10)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			found := false
			for _, r := range results {
				if r.Content == tc.content {
					found = true
					break
				}
			}

			if found != tc.expected {
				t.Errorf("Test %s: expected found=%v, got found=%v", tc.name, tc.expected, found)
				t.Errorf("  Content: %s", tc.content)
				t.Errorf("  Query: %s", tc.query)
				if len(results) > 0 {
					t.Errorf("  Results: %+v", results)
				}
			}
		})
	}
}

// TestGSESegmenter_Direct tests the GSE segmenter directly
func TestGSESegmenter_Direct(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GSE segmenter test in short mode")
	}

	seg, err := NewGSESegmenter()
	if err != nil {
		t.Fatalf("Failed to create GSE segmenter: %v", err)
	}

	testCases := []struct {
		input    string
		minWords int // minimum expected number of segments
	}{
		{"用户喜欢简洁的代码风格", 3},
		{"人工智能技术发展迅速", 3},
		{"使用React框架构建前端应用", 2},
		{"数据库连接池配置优化", 3},
		{"今天天气很好", 2},
	}

	for _, tc := range testCases {
		segments := seg.Cut(tc.input, true)
		t.Logf("Input: %s", tc.input)
		t.Logf("Segments: %v (count: %d)", segments, len(segments))

		if len(segments) < tc.minWords {
			t.Errorf("Expected at least %d segments for '%s', got %d: %v",
				tc.minWords, tc.input, len(segments), segments)
		}
	}
}

// TestChineseSegmenter_SmartSegment tests the smart segment function
func TestChineseSegmenter_SmartSegment(t *testing.T) {
	// Test with Chinese disabled
	csDisabled := NewChineseSegmenter(false)
	text := "用户喜欢简洁的代码风格"
	result := csDisabled.SmartSegment(text)
	if result != text {
		t.Errorf("Expected unchanged text when Chinese disabled, got: %s", result)
	}

	// Test Chinese detection
	detectionTests := []struct {
		text     string
		expected bool
	}{
		{"Hello World", false},
		{"你好世界", true},
		{"Hello 世界", true},
		{"123456", false},
		{"用户喜欢简洁的代码风格", true},
		{"React框架", true},
		{"API", false},
		{"", false},
	}

	for _, tc := range detectionTests {
		result := ContainsChinese(tc.text)
		if result != tc.expected {
			t.Errorf("ContainsChinese(%q) = %v, expected %v", tc.text, result, tc.expected)
		}
	}
}

// BenchmarkChineseSegmentation benchmarks Chinese segmentation performance
func BenchmarkChineseSegmentation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	seg, err := NewGSESegmenter()
	if err != nil {
		b.Fatalf("Failed to create segmenter: %v", err)
	}

	testTexts := []string{
		"用户喜欢简洁的代码风格，并且希望代码有良好的注释",
		"人工智能技术在近年来取得了显著的进步",
		"使用React框架构建前端应用是一种常见的选择",
		"数据库连接池的配置对于应用性能至关重要",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, text := range testTexts {
			seg.Cut(text, true)
		}
	}
}

// BenchmarkChineseSegmentationParallel benchmarks parallel segmentation
func BenchmarkChineseSegmentationParallel(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	seg, err := NewGSESegmenter()
	if err != nil {
		b.Fatalf("Failed to create segmenter: %v", err)
	}

	testText := "用户喜欢简洁的代码风格，并且希望代码有良好的注释"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			seg.Cut(testText, true)
		}
	})
}
