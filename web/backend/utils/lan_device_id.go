package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const lanDeviceIDFileName = ".moonhub_lan_device_id"

// EnsureLANDeviceID returns a stable UUID persisted beside config.json (under configDir).
// Used for mDNS TXT "id=" and LAN APIs. configDir must be non-empty.
func EnsureLANDeviceID(configDir string) (string, error) {
	if configDir == "" {
		return "", fmt.Errorf("config directory is required")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(configDir, lanDeviceIDFileName)
	data, err := os.ReadFile(p)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id, nil
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	id := uuid.NewString()
	if err := os.WriteFile(p, []byte(id), 0o600); err != nil {
		return "", err
	}
	return id, nil
}
