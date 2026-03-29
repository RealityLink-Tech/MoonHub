package fileutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateSafePath checks that a path does not contain directory traversal
// components after cleaning. It returns an error if the cleaned path contains "..".
func ValidateSafePath(path string) error {
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path contains traversal component: %s", cleaned)
	}
	return nil
}
