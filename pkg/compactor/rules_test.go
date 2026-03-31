// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"strings"
	"testing"
)

func TestNormalizeCJKPunctuation(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"你好，世界", "你好,世界"},
		{"这是测试。", "这是测试."},
		{"问题？", "问题?"},
		{"感叹！", "感叹!"},
		{"（括号）", "(括号)"},
		{"【方括号】", "[方括号]"},
		{"分号；", "分号;"},
		{"冒号：", "冒号:"},
		{"混合Mixed，内容。", "混合Mixed,内容."},
		{"英文english unchanged", "英文english unchanged"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeCJKPunctuation(tt.input)
		if got != tt.expect {
			t.Errorf("NormalizeCJKPunctuation(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestStripEmoji(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // check result contains this (or is empty if input was empty)
	}{
		{"simple text", "Hello world", "Hello world"},
		{"emoji removal", "Hello 😊 world", "Hello world"},
		{"multiple emoji", "🎉🎊🎁 Party time", "Party time"},
		{"emoji only", "😀😀😀", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripEmoji(tt.input)
			if tt.contains == "" && got != "" {
				t.Errorf("StripEmoji(%q) = %q, want empty", tt.input, got)
			}
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("StripEmoji(%q) = %q, want to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestDeduplicateLines(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect int // number of lines in result
	}{
		{
			name:   "no duplicates",
			input:  "line 1\nline 2\nline 3",
			expect: 3,
		},
		{
			name:   "exact duplicates",
			input:  "duplicate\nduplicate\nduplicate",
			expect: 1,
		},
		{
			name:   "mixed duplicates",
			input:  "unique 1\nduplicate\nduplicate\nunique 2",
			expect: 3,
		},
		{
			name:   "preserve empty lines",
			input:  "line 1\n\nline 2\n\nline 3",
			expect: 5, // empty lines preserved
		},
		{
			name:   "empty input",
			input:  "",
			expect: 1, // empty string splits to [""]
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeduplicateLines(tt.input)
			lines := len(strings.Split(got, "\n"))
			if lines != tt.expect {
				t.Errorf("DeduplicateLines got %d lines, want %d", lines, tt.expect)
			}
		})
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "collapse multiple newlines",
			input: "line 1\n\n\n\n\nline 2",
			check: func(s string) bool { return !strings.Contains(s, "\n\n\n") },
		},
		{
			name:  "trim trailing spaces",
			input: "line with spaces   \nanother line\t",
			check: func(s string) bool { return !strings.Contains(s, "   \n") },
		},
		{
			name:  "preserve single blank lines",
			input: "line 1\n\nline 2",
			check: func(s string) bool { return strings.Contains(s, "\n\n") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollapseWhitespace(tt.input)
			if !tt.check(got) {
				t.Errorf("CollapseWhitespace check failed, got: %q", got)
			}
		})
	}
}

func TestRemoveDecorativeLines(t *testing.T) {
	input := `Header
---
Content here
***
More content
===
Final`

	result := RemoveDecorativeLines(input)

	if strings.Contains(result, "---") {
		t.Error("Result should not contain '---'")
	}
	if strings.Contains(result, "***") {
		t.Error("Result should not contain '***'")
	}
	if strings.Contains(result, "===") {
		t.Error("Result should not contain '==='")
	}
	if !strings.Contains(result, "Header") {
		t.Error("Result should contain 'Header'")
	}
}

func TestRemoveEmptySections(t *testing.T) {
	input := `# Section With Content
Some content here.

# Empty Section

# Another Section
More content.`

	result := RemoveEmptySections(input)

	if strings.Contains(result, "Empty Section") {
		t.Error("Result should not contain empty section header")
	}
	if !strings.Contains(result, "Section With Content") {
		t.Error("Result should contain non-empty section")
	}
	if !strings.Contains(result, "Another Section") {
		t.Error("Result should contain section with content")
	}
}

func TestCompressMarkdownTable(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string // substring to find in output
	}{
		{
			name: "2-column table becomes key:value",
			input: `| Key | Value |
|-----|-------|
| name | Alice |
| age | 30 |`,
			expect: "- name: Alice",
		},
		{
			name: "non-table text preserved",
			input: `Some text before.

| A | B |
|---|---|
| 1 | 2 |

Some text after.`,
			expect: "Some text before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompressMarkdownTable(tt.input)
			if !strings.Contains(got, tt.expect) {
				t.Errorf("CompressMarkdownTable result should contain %q, got: %q", tt.expect, got)
			}
		})
	}
}

func TestHasCJK(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Hello world", false},
		{"你好世界", true},
		{"Hello 世界", true},
		{"日本語", true},
		{"한국어", true},
		{"", false},
		{"123", false},
	}

	for _, tt := range tests {
		got := HasCJK(tt.input)
		if got != tt.want {
			t.Errorf("HasCJK(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestHasTables(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "has table",
			input: "| A | B |\n|---|---|\n| 1 | 2 |",
			want:  true,
		},
		{
			name:  "no table",
			input: "Just regular text\nNo table here",
			want:  false,
		},
		{
			name:  "pipe but no separator",
			input: "Text with | pipe\nNo separator",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTables(tt.input)
			if got != tt.want {
				t.Errorf("HasTables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSmartRuleSelection(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		check    func(PreCompressOptions) bool
	}{
		{
			name:     "CJK content enables normalization",
			messages: []string{"这是中文内容"},
			check:    func(o PreCompressOptions) bool { return o.NormalizeCJK },
		},
		{
			name:     "table content enables compression",
			messages: []string{"| A | B |\n|---|---|\n| 1 | 2 |"},
			check:    func(o PreCompressOptions) bool { return o.CompressTables },
		},
		{
			name:     "plain text keeps defaults",
			messages: []string{"Just plain English text"},
			check:    func(o PreCompressOptions) bool { return !o.NormalizeCJK && !o.CompressTables },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SmartRuleSelection(tt.messages)
			if !tt.check(got) {
				t.Errorf("SmartRuleSelection check failed for %s: %+v", tt.name, got)
			}
		})
	}
}

func TestPreCompress(t *testing.T) {
	opts := PreCompressOptions{
		StripEmoji:           true,
		RemoveDuplicateLines: true,
		NormalizeCJK:         true,
		CompressTables:       true,
		MergeBullets:         true,
	}

	input := `# Header
这是测试内容，包含中文。

Duplicate line.
Duplicate line.

😊

- Short item 1
- Short item 2
- Short item 3

---
`

	result := PreCompress(input, opts)

	// Should not contain emoji
	if strings.Contains(result, "😊") {
		t.Error("PreCompress should strip emoji")
	}

	// Should have CJK punctuation normalized
	if strings.Contains(result, "，") {
		t.Error("PreCompress should normalize CJK punctuation")
	}

	// Result should be shorter than input
	if len(result) >= len(input) {
		t.Errorf("PreCompress result (%d) should be shorter than input (%d)", len(result), len(input))
	}

	// Should not contain duplicate lines (the "Duplicate line." appears twice)
	lines := strings.Split(result, "\n")
	duplicateCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "Duplicate line." {
			duplicateCount++
		}
	}
	if duplicateCount > 1 {
		t.Error("PreCompress should remove duplicate lines")
	}
}

func TestSimilarityRatio(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64
		max  float64
	}{
		{"identical", "identical", 1.0, 1.0},
		{"short", "", 0.0, 0.0},
		{"hello world", "hello world", 1.0, 1.0},
		{"hello world", "hello earth", 0.5, 0.9},
		{"completely different", "totally unlike", 0.0, 0.3},
	}

	for _, tt := range tests {
		got := similarityRatio(tt.a, tt.b)
		if got < tt.min || got > tt.max {
			t.Errorf("similarityRatio(%q, %q) = %f, want between %f and %f",
				tt.a, tt.b, got, tt.min, tt.max)
		}
	}
}

func TestMergeSimilarBullets(t *testing.T) {
	input := `- This is a test bullet with more content here
- This is a test bullet with more content here too  // very similar
- Another unique item
- Different content here`

	result := MergeSimilarBullets(input, 0.7)

	// Very similar bullets should be merged (one removed)
	// Result should have fewer lines
	originalLines := len(strings.Split(input, "\n"))
	resultLines := len(strings.Split(result, "\n"))

	// At least one bullet should be merged
	if resultLines >= originalLines {
		t.Errorf("MergeSimilarBullets should reduce bullet count, got %d from %d",
			resultLines, originalLines)
	}
}

func TestMergeShortBullets(t *testing.T) {
	input := `- a
- b
- c
- d
- This is a longer bullet with many words`

	result := MergeShortBullets(input, 3, 5)

	// Short bullets should be merged into comma-separated form
	if !strings.Contains(result, ", ") {
		t.Errorf("MergeShortBullets should create comma-separated list, got: %q", result)
	}

	// Longer bullet should be preserved as-is
	if !strings.Contains(result, "longer bullet") {
		t.Errorf("MergeShortBullets should preserve longer bullets, got: %q", result)
	}
}
