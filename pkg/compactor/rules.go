// MoonHub - Your ready-to-use AI assistant
// Copyright (c) 2026 MoonHub contributors
// License: MIT

package compactor

import (
	"regexp"
	"sort"
	"strings"
)

// PreCompressOptions configures which rules to apply
type PreCompressOptions struct {
	StripEmoji           bool
	RemoveDuplicateLines bool
	NormalizeCJK         bool
	CompressTables       bool
	MergeBullets         bool
}

// Chinese fullwidth punctuation -> ASCII equivalents (each saves ~1 token)
var zhPunctMap = map[rune]rune{
	'，':      ',',
	'。':      '.',
	'；':      ';',
	'：':      ':',
	'！':      '!',
	'？':      '?',
	'\u201C': '"',  // " left double quotation mark
	'\u201D': '"',  // " right double quotation mark
	'\u2018': '\'', // ' left single quotation mark
	'\u2019': '\'', // ' right single quotation mark
	'（':      '(',
	'）':      ')',
	'【':      '[',
	'】':      ']',
	'、':      ',',
	'…':      '.', // Simplified
	'~':      '~',
}

// Matches most emoji: emoticons, dingbats, symbols, skin tones, ZWJ sequences
var emojiRegex = regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F1E0}-\x{1F1FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FE0F}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{200D}\x{20E3}\x{E0020}-\x{E007F}]`)

// Markdown header regex
var headerRE = regexp.MustCompile(`^(#{1,6})\s+(.*)`)

// Table separator line regex
var tableSepRE = regexp.MustCompile(`^[\s|:-]+$`)

// Bullet line regex
var bulletRE = regexp.MustCompile(`^(\s*[-*+]\s+)(.*)`)

// Decorative lines regex (---, ***, ===)
var decorativeRE = regexp.MustCompile(`^[\s]*[-*=]{3,}[\s]*$`)

// NormalizeCJKPunctuation normalizes Chinese fullwidth punctuation to ASCII equivalents.
// Each replacement typically saves ~1 token.
func NormalizeCJKPunctuation(text string) string {
	if text == "" {
		return ""
	}

	// Handle em-dash pair
	text = strings.ReplaceAll(text, "——", "--")

	// Replace Chinese punctuation
	result := strings.Builder{}
	result.Grow(len(text))

	for _, r := range text {
		if replacement, ok := zhPunctMap[r]; ok {
			result.WriteRune(replacement)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// StripEmoji removes emoji characters from text.
func StripEmoji(text string) string {
	text = emojiRegex.ReplaceAllString(text, "")
	text = regexp.MustCompile(`\s{2,}`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// DeduplicateLines removes exact duplicate lines while preserving order.
// Keeps the first occurrence of each line.
func DeduplicateLines(text string) string {
	lines := strings.Split(text, "\n")
	seen := make(map[string]bool)
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Keep empty lines for structure, deduplicate non-empty ones
		if trimmed == "" || !seen[trimmed] {
			result = append(result, line)
			if trimmed != "" {
				seen[trimmed] = true
			}
		}
	}

	return strings.Join(result, "\n")
}

// CollapseWhitespace collapses runs of 3+ blank lines into a single blank line.
// Also trims trailing whitespace from each line.
func CollapseWhitespace(text string) string {
	lines := strings.Split(text, "\n")

	// Trim trailing whitespace from each line
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	result := strings.Join(lines, "\n")

	// Collapse 3+ newlines into 2
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return result
}

// RemoveDecorativeLines removes lines that are entirely empty content markers like "---", "***", "===".
func RemoveDecorativeLines(text string) string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if !decorativeRE.MatchString(line) {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// RemoveEmptySections removes markdown sections that have no meaningful body content.
// A section with only whitespace and no child sections is removed.
func RemoveEmptySections(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")

	type Section struct {
		header string
		level  int
		body   []string
	}

	var sections []Section
	current := Section{header: "", level: 0, body: []string{}}

	for _, line := range lines {
		matches := headerRE.FindStringSubmatch(line)
		if matches != nil {
			sections = append(sections, current)
			level := len(matches[1])
			current = Section{
				header: strings.TrimSpace(matches[2]),
				level:  level,
				body:   []string{},
			}
		} else {
			current.body = append(current.body, line)
		}
	}
	sections = append(sections, current)

	// Determine which sections have children (a deeper section follows)
	hasChild := make([]bool, len(sections))
	for i := 1; i < len(sections); i++ {
		if sections[i].level > 0 {
			for p := i - 1; p >= 0; p-- {
				if sections[p].level > 0 && sections[p].level < sections[i].level {
					hasChild[p] = true
					break
				}
			}
		}
	}

	var result []string
	for i, sec := range sections {
		body := strings.TrimSpace(strings.Join(sec.body, "\n"))

		if sec.header == "" && body == "" {
			continue
		}
		if sec.header != "" && body == "" && !hasChild[i] {
			continue // empty section, no children
		}

		if sec.header != "" {
			result = append(result, strings.Repeat("#", sec.level)+" "+sec.header)
		}
		if body != "" {
			result = append(result, body)
		}
		result = append(result, "") // blank line between sections
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// CompressMarkdownTable converts markdown tables to compact key:value notation.
// 2-column tables become "- Key: Value" lines.
// Multi-column tables become compact "Col1, Header2=Val2" lines.
// Wide tables (5+ columns) are preserved as pipe-delimited rows without header/separator.
func CompressMarkdownTable(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var result []string
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Check if this is a table header followed by separator
		if strings.Contains(line, "|") && i+1 < len(lines) && tableSepRE.MatchString(strings.TrimSpace(lines[i+1])) {
			// Parse headers
			headers := parseTableRow(line)

			i += 2 // skip header + separator

			// Parse rows
			var rows [][]string
			for i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != "" {
				rows = append(rows, parseTableRow(lines[i]))
				i++
			}

			// Compress table
			if len(headers) >= 5 {
				// Wide tables: preserve rows without header/separator
				for _, row := range rows {
					result = append(result, "| "+strings.Join(row, " | ")+" |")
				}
			} else if len(headers) == 2 {
				// 2-column: key: value format
				for _, row := range rows {
					if len(row) >= 2 {
						result = append(result, "- "+row[0]+": "+row[1])
					} else if len(row) == 1 && row[0] != "" {
						result = append(result, "- "+row[0]+": ")
					}
				}
			} else {
				// Multi-column: compact format using headers as labels
				for _, row := range rows {
					var parts []string
					for ci := 0; ci < len(row); ci++ {
						if ci == 0 {
							parts = append(parts, row[ci])
						} else if ci < len(headers) {
							parts = append(parts, headers[ci]+"="+row[ci])
						} else {
							parts = append(parts, row[ci])
						}
					}
					result = append(result, strings.Join(parts, ", "))
				}
			}
		} else {
			result = append(result, line)
			i++
		}
	}

	return strings.Join(result, "\n")
}

// parseTableRow parses a markdown table row into cells
func parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	parts := strings.Split(trimmed, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

// similarityRatio computes similarity ratio between two strings (0.0-1.0).
// Simplified SequenceMatcher-style comparison using bigram overlap.
func similarityRatio(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}

	// Compute bigrams
	bigramsA := make(map[string]int)
	for i := 0; i < len(a)-1; i++ {
		bg := a[i : i+2]
		bigramsA[bg]++
	}

	bigramsB := make(map[string]int)
	for i := 0; i < len(b)-1; i++ {
		bg := b[i : i+2]
		bigramsB[bg]++
	}

	// Compute intersection
	intersection := 0
	for bg, countA := range bigramsA {
		if countB, ok := bigramsB[bg]; ok {
			intersection += min(countA, countB)
		}
	}

	total := (len(a) - 1) + (len(b) - 1)
	if total == 0 {
		return 0
	}
	return float64(2*intersection) / float64(total)
}

// MergeSimilarBullets merges bullet lines with high similarity (>= threshold).
// When two bullets are near-identical, keep the longer one.
func MergeSimilarBullets(text string, threshold float64) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var result []string

	type bullet struct {
		prefix  string
		content string
		line    string
	}
	var bullets []bullet

	flushBullets := func() {
		if len(bullets) == 0 {
			return
		}

		mergedOut := make([]bool, len(bullets))
		for i := 0; i < len(bullets); i++ {
			if mergedOut[i] {
				continue
			}
			for j := i + 1; j < len(bullets); j++ {
				if mergedOut[j] {
					continue
				}
				ratio := similarityRatio(bullets[i].content, bullets[j].content)
				if ratio >= threshold {
					// Keep the longer one
					if len(bullets[j].content) > len(bullets[i].content) {
						mergedOut[i] = true
						break
					} else {
						mergedOut[j] = true
					}
				}
			}
		}

		for i, b := range bullets {
			if !mergedOut[i] {
				result = append(result, b.line)
			}
		}
		bullets = bullets[:0]
	}

	for _, line := range lines {
		matches := bulletRE.FindStringSubmatch(line)
		if matches != nil {
			bullets = append(bullets, bullet{
				prefix:  matches[1],
				content: matches[2],
				line:    line,
			})
		} else {
			flushBullets()
			result = append(result, line)
		}
	}
	flushBullets()

	return strings.Join(result, "\n")
}

// MergeShortBullets combines consecutive short bullet points into comma-separated form.
// Bullets with <= maxWords words are candidates. Up to maxMerge consecutive short bullets are joined.
func MergeShortBullets(text string, maxWords, maxMerge int) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var result []string
	var shortBullets []string
	bulletPrefix := "- "

	flushShort := func() {
		if len(shortBullets) == 0 {
			return
		}
		if len(shortBullets) <= 2 {
			for _, sb := range shortBullets {
				result = append(result, bulletPrefix+sb)
			}
		} else {
			result = append(result, bulletPrefix+strings.Join(shortBullets, ", "))
		}
		shortBullets = shortBullets[:0]
	}

	for _, line := range lines {
		matches := bulletRE.FindStringSubmatch(line)
		if matches != nil {
			content := strings.TrimSpace(matches[2])
			bulletPrefix = matches[1]

			if content == "" {
				flushShort()
				result = append(result, line)
			} else if len(strings.Fields(content)) <= maxWords {
				shortBullets = append(shortBullets, content)
				if len(shortBullets) >= maxMerge {
					flushShort()
				}
			} else {
				flushShort()
				result = append(result, line)
			}
		} else {
			flushShort()
			result = append(result, line)
		}
	}
	flushShort()

	return strings.Join(result, "\n")
}

// HasCJK checks if text contains CJK characters
func HasCJK(text string) bool {
	for _, r := range text {
		if isCJK(r) {
			return true
		}
	}
	return false
}

// HasTables checks if text contains markdown tables
func HasTables(text string) bool {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.Contains(line, "|") && i+1 < len(lines) && tableSepRE.MatchString(strings.TrimSpace(lines[i+1])) {
			return true
		}
	}
	return false
}

// SmartRuleSelection dynamically chooses rules based on content analysis
func SmartRuleSelection(messages []string) PreCompressOptions {
	opts := PreCompressOptions{
		StripEmoji:           true,
		RemoveDuplicateLines: true,
		NormalizeCJK:         false,
		CompressTables:       false,
		MergeBullets:         true,
	}

	// Analyze content to determine which rules to apply
	for _, m := range messages {
		if HasCJK(m) {
			opts.NormalizeCJK = true
		}
		if HasTables(m) {
			opts.CompressTables = true
		}
	}

	return opts
}

// PreCompress applies all pre-compression rules to a message's content.
// The full 9-rule pipeline runs in this order:
//  1. CJK punctuation normalization
//  2. Collapse whitespace
//  3. Deduplicate lines
//  4. Remove empty sections
//  5. Compress markdown tables
//  6. Strip emoji (if enabled)
//  7. Merge similar bullets
//  8. Merge short bullets
//  9. Remove decorative lines + final cleanup
func PreCompress(text string, opts PreCompressOptions) string {
	result := text

	// 1. CJK punctuation normalization
	if opts.NormalizeCJK {
		result = NormalizeCJKPunctuation(result)
	}

	// 2. Collapse whitespace
	result = CollapseWhitespace(result)

	// 3. Deduplicate lines
	if opts.RemoveDuplicateLines {
		result = DeduplicateLines(result)
	}

	// 4. Remove empty sections
	result = RemoveEmptySections(result)

	// 5. Compress markdown tables
	if opts.CompressTables {
		result = CompressMarkdownTable(result)
	}

	// 6. Strip emoji
	if opts.StripEmoji {
		result = StripEmoji(result)
	}

	// 7. Merge similar bullets
	if opts.MergeBullets {
		result = MergeSimilarBullets(result, 0.8)
	}

	// 8. Merge short bullets
	result = MergeShortBullets(result, 3, 10)

	// 9. Final cleanup
	result = CollapseWhitespace(result)
	result = RemoveDecorativeLines(result)

	return result
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RuleStats tracks which rules were applied
type RuleStats struct {
	HasCJK        bool
	HasTables     bool
	HasEmoji      bool
	HasDuplicates bool
}

// AnalyzeContent analyzes content to determine rule applicability
func AnalyzeContent(messages []string) RuleStats {
	stats := RuleStats{}
	for _, m := range messages {
		if HasCJK(m) {
			stats.HasCJK = true
		}
		if HasTables(m) {
			stats.HasTables = true
		}
		if emojiRegex.MatchString(m) {
			stats.HasEmoji = true
		}
		// Check for duplicate lines
		lines := strings.Split(m, "\n")
		seen := make(map[string]bool)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if seen[trimmed] {
					stats.HasDuplicates = true
					break
				}
				seen[trimmed] = true
			}
		}
	}
	return stats
}

// GetAppliedRules returns a list of rule names that were applied based on options
func GetAppliedRules(opts PreCompressOptions, stats RuleStats) []string {
	var applied []string

	if opts.NormalizeCJK && stats.HasCJK {
		applied = append(applied, "cjk_normalization")
	}
	applied = append(applied, "collapse_whitespace")

	if opts.RemoveDuplicateLines && stats.HasDuplicates {
		applied = append(applied, "deduplicate_lines")
	}

	applied = append(applied, "remove_empty_sections")

	if opts.CompressTables && stats.HasTables {
		applied = append(applied, "compress_tables")
	}

	if opts.StripEmoji && stats.HasEmoji {
		applied = append(applied, "strip_emoji")
	}

	if opts.MergeBullets {
		applied = append(applied, "merge_similar_bullets", "merge_short_bullets")
	}

	applied = append(applied, "remove_decorative_lines")

	// Sort for consistent output
	sort.Strings(applied)
	return applied
}
