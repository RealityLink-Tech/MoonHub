package devices

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

const (
	// CodeExpiry is how long a pairing code is valid.
	CodeExpiry = 5 * time.Minute
)

// PairingManager manages pairing codes for device authentication.
type PairingManager struct {
	mu          sync.RWMutex
	current     *PairingCode
	usedCodes   map[string]*PairingCode // track used codes for potential regeneration
	deviceStore *DeviceStore
}

// NewPairingManager creates a new pairing manager.
func NewPairingManager(deviceStore *DeviceStore) *PairingManager {
	return &PairingManager{
		usedCodes:   make(map[string]*PairingCode),
		deviceStore: deviceStore,
	}
}

// GenerateCodeOnce generates a single pairing code that persists until used.
// This should be called once during device initialization.
func (pm *PairingManager) GenerateCodeOnce() (string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Return existing code if still valid
	if pm.current != nil && pm.current.UsedAt == nil {
		// Check if code is expired
		if time.Since(pm.current.CreatedAt) < CodeExpiry {
			return pm.current.Code, nil
		}
	}

	// Generate new code
	code, err := generatePairingCode()
	if err != nil {
		return "", err
	}

	pm.current = &PairingCode{
		Code:      code,
		CreatedAt: time.Now(),
	}

	return code, nil
}

// GetCurrentCode returns the current pairing code without generating a new one.
// Returns empty string if no code exists or code is used.
func (pm *PairingManager) GetCurrentCode() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.current == nil || pm.current.UsedAt != nil {
		return ""
	}

	// Check if code is expired
	if time.Since(pm.current.CreatedAt) >= CodeExpiry {
		return ""
	}

	return pm.current.Code
}

// ValidateCode validates a pairing code and marks it as used.
// Returns the pairing result with token if successful.
func (pm *PairingManager) ValidateCode(code string, deviceInfo *PairedDevice) (*PairingResult, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if we have a current code
	if pm.current == nil {
		return &PairingResult{
			Success: false,
			Error:   "no pairing code available",
		}, nil
	}

	// Check if code matches
	if pm.current.Code != code {
		return &PairingResult{
			Success: false,
			Error:   "invalid pairing code",
		}, nil
	}

	// Check if code is already used
	if pm.current.UsedAt != nil {
		return &PairingResult{
			Success: false,
			Error:   "pairing code already used",
		}, nil
	}

	// Check if code is expired
	if time.Since(pm.current.CreatedAt) >= CodeExpiry {
		return &PairingResult{
			Success: false,
			Error:   "pairing code expired",
		}, nil
	}

	// Generate token for the device
	token, expiresAt, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Set device info
	now := time.Now()
	deviceInfo.Token = token
	deviceInfo.TokenExpiresAt = expiresAt
	deviceInfo.PairedAt = now
	deviceInfo.LastSeenAt = now

	// Store the paired device
	if err := pm.deviceStore.AddDevice(deviceInfo); err != nil {
		return nil, fmt.Errorf("failed to store paired device: %w", err)
	}

	// Mark code as used
	pm.current.UsedAt = &now
	pm.current.PairedWith = deviceInfo.ID
	pm.usedCodes[code] = pm.current

	return &PairingResult{
		Success:        true,
		Token:          token,
		TokenExpiresAt: expiresAt,
		Device:         deviceInfo,
	}, nil
}

// IsPaired returns true if at least one device is paired.
func (pm *PairingManager) IsPaired() bool {
	return pm.deviceStore.HasDevices()
}

// RegenerateCode generates a new pairing code, invalidating the old one.
func (pm *PairingManager) RegenerateCode() (string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Move current to used if exists
	if pm.current != nil && pm.current.UsedAt == nil {
		now := time.Now()
		pm.current.UsedAt = &now
		pm.current.PairedWith = "regenerated"
		pm.usedCodes[pm.current.Code] = pm.current
	}

	// Generate new code
	code, err := generatePairingCode()
	if err != nil {
		return "", err
	}

	pm.current = &PairingCode{
		Code:      code,
		CreatedAt: time.Now(),
	}

	return code, nil
}

// generatePairingCode generates a random 6-character pairing code.
// Format: 2 letters + 4 digits (e.g., "XM8888", "AB1234").
func generatePairingCode() (string, error) {
	// Generate 2 random letters (A-Z)
	var letterBytes [2]byte
	if _, err := rand.Read(letterBytes[:]); err != nil {
		return "", err
	}

	letters := make([]byte, 2)
	for i, b := range letterBytes {
		letters[i] = 'A' + (b % 26)
	}

	// Generate 4 random digits
	var digitBytes [4]byte
	if _, err := rand.Read(digitBytes[:]); err != nil {
		return "", err
	}

	n := binary.BigEndian.Uint32(digitBytes[:]) % 10000

	return fmt.Sprintf("%s%04d", string(letters), n), nil
}
