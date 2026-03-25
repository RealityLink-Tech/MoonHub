package devices

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewDeviceStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewDeviceStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	if store == nil {
		t.Fatal("expected store instance")
	}

	// Check file path
	expectedPath := filepath.Join(tmpDir, StoreFileName)
	if store.path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, store.path)
	}
}

func TestAddAndGetDevice(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          "test-token-123",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
		LastSeenAt:     time.Now(),
	}

	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Get device by ID
	retrieved := store.GetDevice("test-device-001")
	if retrieved == nil {
		t.Fatal("expected to retrieve device")
	}

	if retrieved.Name != "Test Device" {
		t.Errorf("expected name Test Device, got %s", retrieved.Name)
	}
}

func TestGetDeviceByToken(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          "test-token-123",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
	}

	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Get device by token
	retrieved := store.GetDeviceByToken("test-token-123")
	if retrieved == nil {
		t.Fatal("expected to retrieve device by token")
	}

	if retrieved.ID != "test-device-001" {
		t.Errorf("expected ID test-device-001, got %s", retrieved.ID)
	}

	// Non-existent token
	notFound := store.GetDeviceByToken("invalid-token")
	if notFound != nil {
		t.Error("expected nil for invalid token")
	}
}

func TestRemoveDevice(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          "test-token-123",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
	}

	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Remove device
	if err := store.RemoveDevice("test-device-001"); err != nil {
		t.Fatalf("failed to remove device: %v", err)
	}

	// Verify removed
	retrieved := store.GetDevice("test-device-001")
	if retrieved != nil {
		t.Error("expected device to be removed")
	}
}

func TestValidateToken(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	// Valid token
	token, expiresAt, err := GenerateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          token,
		TokenExpiresAt: expiresAt,
		PairedAt:       time.Now(),
	}

	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Validate token
	validation := store.ValidateToken(token)
	if !validation.Valid {
		t.Errorf("expected token to be valid, got error: %s", validation.Error)
	}

	if validation.DeviceID != "test-device-001" {
		t.Errorf("expected device ID test-device-001, got %s", validation.DeviceID)
	}

	// Invalid token
	invalidValidation := store.ValidateToken("invalid-token")
	if invalidValidation.Valid {
		t.Error("expected invalid token to be invalid")
	}
}

func TestValidateExpiredToken(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	// Generate a valid token format
	token, _, err := GenerateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Create device with expired token
	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          token,
		TokenExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		PairedAt:       time.Now(),
	}

	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Validate expired token
	validation := store.ValidateToken(token)
	if validation.Valid {
		t.Error("expected expired token to be invalid")
	}

	if validation.Error != "token expired" {
		t.Errorf("expected 'token expired' error, got: %s", validation.Error)
	}
}

func TestListDevices(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	// Add multiple devices
	for i := 1; i <= 3; i++ {
		device := &PairedDevice{
			ID:             string(rune('0' + i)),
			Name:           "Test Device",
			Token:          string(rune('a' + i)),
			TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
			PairedAt:       time.Now(),
		}
		if err := store.AddDevice(device); err != nil {
			t.Fatalf("failed to add device: %v", err)
		}
	}

	devices := store.ListDevices()
	if len(devices) != 3 {
		t.Errorf("expected 3 devices, got %d", len(devices))
	}
}

func TestHasDevices(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	// No devices initially
	if store.HasDevices() {
		t.Error("expected no devices initially")
	}

	// Add device
	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          "test-token",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
	}
	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Now has devices
	if !store.HasDevices() {
		t.Error("expected to have devices")
	}
}

func TestUpdateLastSeen(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          "test-token",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
		LastSeenAt:     time.Now().Add(-1 * time.Hour),
	}
	if err := store.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Update last seen
	if err := store.UpdateLastSeen("test-device-001", "192.168.1.100"); err != nil {
		t.Fatalf("failed to update last seen: %v", err)
	}

	// Verify update
	updated := store.GetDevice("test-device-001")
	if updated.IPAddress != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", updated.IPAddress)
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	// Add device with expired token
	expiredDevice := &PairedDevice{
		ID:             "expired-device",
		Name:           "Expired Device",
		Token:          "expired-token",
		TokenExpiresAt: time.Now().Add(-1 * time.Hour),
		PairedAt:       time.Now(),
	}
	if err := store.AddDevice(expiredDevice); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Add device with valid token
	validDevice := &PairedDevice{
		ID:             "valid-device",
		Name:           "Valid Device",
		Token:          "valid-token",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
	}
	if err := store.AddDevice(validDevice); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Cleanup
	removed := store.CleanupExpiredTokens()
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	// Verify
	if store.HasDevice("expired-device") {
		t.Error("expected expired device to be removed")
	}
	if !store.HasDevice("valid-device") {
		t.Error("expected valid device to still exist")
	}
}

func TestPersistAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and add device
	store1, err := NewDeviceStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	device := &PairedDevice{
		ID:             "test-device-001",
		Name:           "Test Device",
		Token:          "test-token",
		TokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		PairedAt:       time.Now(),
	}
	if err := store1.AddDevice(device); err != nil {
		t.Fatalf("failed to add device: %v", err)
	}

	// Create new store (should load from file)
	store2, err := NewDeviceStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create second store: %v", err)
	}

	// Verify data persisted
	retrieved := store2.GetDevice("test-device-001")
	if retrieved == nil {
		t.Fatal("expected device to persist")
	}

	if retrieved.Name != "Test Device" {
		t.Errorf("expected name Test Device, got %s", retrieved.Name)
	}
}
