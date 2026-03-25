package devices

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/fileutil"
)

const (
	// StoreFileName is the name of the paired devices store file.
	StoreFileName = "paired_devices.json"
)

// DeviceStore manages persistent storage of paired devices.
type DeviceStore struct {
	mu      sync.RWMutex
	path    string
	devices map[string]*PairedDevice
}

// NewDeviceStore creates a new device store at <storeDir>/paired_devices.json.
// If storeDir is empty, uses ~/.moonhub (same default location as config.json).
func NewDeviceStore(storeDir string) (*DeviceStore, error) {
	if storeDir == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir == "" {
			homeDir = "."
		}
		storeDir = filepath.Join(homeDir, ".moonhub")
	}

	path := filepath.Join(storeDir, StoreFileName)

	store := &DeviceStore{
		path:    path,
		devices: make(map[string]*PairedDevice),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// load reads the store from disk.
func (s *DeviceStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.devices)
}

// save writes the store to disk.
func (s *DeviceStore) save() error {
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return err
	}

	return fileutil.WriteFileAtomic(s.path, data, 0600)
}

// AddDevice adds or updates a paired device.
func (s *DeviceStore) AddDevice(device *PairedDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.devices[device.ID] = device
	return s.save()
}

// RemoveDevice removes a paired device by ID.
func (s *DeviceStore) RemoveDevice(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.devices, deviceID)
	return s.save()
}

// GetDevice retrieves a paired device by ID.
func (s *DeviceStore) GetDevice(deviceID string) *PairedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.devices[deviceID]
}

// GetDeviceByToken retrieves a paired device by its token.
func (s *DeviceStore) GetDeviceByToken(token string) *PairedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.devices {
		if device.Token == token {
			return device
		}
	}

	return nil
}

// ValidateToken checks if a token is valid and not expired.
func (s *DeviceStore) ValidateToken(token string) *TokenValidation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Basic format check
	if !ValidateToken(token) {
		return &TokenValidation{
			Valid: false,
			Error: "invalid token format",
		}
	}

	// Find device with this token
	for _, device := range s.devices {
		if device.Token == token {
			// Check expiration
			if time.Now().After(device.TokenExpiresAt) {
				return &TokenValidation{
					Valid: false,
					Error: "token expired",
				}
			}

			expiresAt := device.TokenExpiresAt
			return &TokenValidation{
				Valid:     true,
				DeviceID:  device.ID,
				ExpiresAt: &expiresAt,
			}
		}
	}

	return &TokenValidation{
		Valid: false,
		Error: "token not found",
	}
}

// UpdateLastSeen updates the last seen time for a device.
func (s *DeviceStore) UpdateLastSeen(deviceID string, ipAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil
	}

	device.LastSeenAt = time.Now()
	device.IPAddress = ipAddress

	return s.save()
}

// ListDevices returns all paired devices.
func (s *DeviceStore) ListDevices() []*PairedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*PairedDevice, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}

	return devices
}

// HasDevice checks if a specific device exists.
func (s *DeviceStore) HasDevice(deviceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.devices[deviceID]
	return exists
}

// HasDevices returns true if there are any paired devices.
func (s *DeviceStore) HasDevices() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.devices) > 0
}

// Count returns the number of paired devices.
func (s *DeviceStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.devices)
}

// CleanupExpiredTokens removes devices with expired tokens.
func (s *DeviceStore) CleanupExpiredTokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, device := range s.devices {
		if now.After(device.TokenExpiresAt) {
			delete(s.devices, id)
			removed++
		}
	}

	if removed > 0 {
		_ = s.save()
	}

	return removed
}
