// MoonHub - Your ready-to-use AI assistant
// Adaptive Memory System - Migration from MEMORY.md
// License: MIT

package adaptive_memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Migrator handles migration from legacy MEMORY.md to adaptive memory
type Migrator struct {
	engine     *Engine
	memoryPath string
	markPath   string
}

// NewMigrator creates a new migrator
func NewMigrator(engine *Engine, memoryPath string) *Migrator {
	memoryDir := filepath.Dir(memoryPath)
	markPath := filepath.Join(memoryDir, ".migrated")

	return &Migrator{
		engine:     engine,
		memoryPath: memoryPath,
		markPath:   markPath,
	}
}

// IsMigrated checks if migration has already been done
func (m *Migrator) IsMigrated() bool {
	_, err := os.Stat(m.markPath)
	return err == nil
}

// Migrate performs the migration from MEMORY.md to adaptive memory
func (m *Migrator) Migrate() error {
	// Check if already migrated
	if m.IsMigrated() {
		return nil
	}

	// Check if MEMORY.md exists
	if _, err := os.Stat(m.memoryPath); os.IsNotExist(err) {
		return nil // Nothing to migrate
	}

	// Read MEMORY.md
	content, err := os.ReadFile(m.memoryPath)
	if err != nil {
		return fmt.Errorf("failed to read MEMORY.md: %w", err)
	}

	// Parse and import
	records := m.parseMemoryMD(string(content))
	for _, rec := range records {
		_, err := m.engine.RecordEvent("default", EventInput{
			Type:       EventTypeFactStored,
			Content:    rec,
			Importance: floatPtr(0.7), // Slightly higher importance for migrated facts
		})
		if err != nil {
			// Log but continue
			continue
		}
	}

	// Backup original file
	backupPath := m.memoryPath + ".backup"
	if err := os.Rename(m.memoryPath, backupPath); err != nil {
		// If rename fails, try copy
		if err := copyFile(m.memoryPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup MEMORY.md: %w", err)
		}
	}

	// Create migration mark
	if err := m.createMark(); err != nil {
		return fmt.Errorf("failed to create migration mark: %w", err)
	}

	return nil
}

// parseMemoryMD parses MEMORY.md content into individual records
func (m *Migrator) parseMemoryMD(content string) []string {
	var records []string
	var currentSection strings.Builder
	var sectionTitle string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for section headers
		if strings.HasPrefix(line, "## ") {
			// Save previous section if not empty
			if currentSection.Len() > 0 {
				text := strings.TrimSpace(currentSection.String())
				if text != "" && text != sectionTitle {
					records = append(records, m.formatRecord(sectionTitle, text))
				}
			}

			// Start new section
			sectionTitle = strings.TrimPrefix(line, "## ")
			currentSection.Reset()
			continue
		}

		// Check for main headers
		if strings.HasPrefix(line, "# ") {
			// Save previous section if not empty
			if currentSection.Len() > 0 && sectionTitle != "" {
				text := strings.TrimSpace(currentSection.String())
				if text != "" && text != sectionTitle {
					records = append(records, m.formatRecord(sectionTitle, text))
				}
			}
			sectionTitle = ""
			currentSection.Reset()
			continue
		}

		// Add line to current section
		if sectionTitle != "" {
			currentSection.WriteString(line)
			currentSection.WriteString("\n")
		}
	}

	// Handle last section
	if currentSection.Len() > 0 && sectionTitle != "" {
		text := strings.TrimSpace(currentSection.String())
		if text != "" && text != sectionTitle {
			records = append(records, m.formatRecord(sectionTitle, text))
		}
	}

	return records
}

// formatRecord formats a section as a memory record
func (m *Migrator) formatRecord(title, content string) string {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if content == "" {
		return title
	}

	return fmt.Sprintf("[%s] %s", title, content)
}

// createMark creates the migration mark file
func (m *Migrator) createMark() error {
	return os.WriteFile(m.markPath, []byte("migrated"), 0644)
}

// copyFile copies a file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// floatPtr returns a pointer to a float64
func floatPtr(v float64) *float64 {
	return &v
}
