package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// mockDeviceStore is a mock implementation of the device store for testing.
type mockDeviceStore struct {
	devices []*devices.PairedDevice
}

func (m *mockDeviceStore) ListDevices() []*devices.PairedDevice {
	return m.devices
}

func (m *mockDeviceStore) GetDevice(deviceID string) *devices.PairedDevice {
	for _, device := range m.devices {
		if device.ID == deviceID {
			return device
		}
	}
	return nil
}

// newTestDevicesHandler creates a DevicesHandler with a mock store.
func newTestDevicesHandler(devices []*devices.PairedDevice) *DevicesHandler {
	return &DevicesHandler{
		deviceStore: &mockDeviceStore{devices: devices},
	}
}

func TestDevicesHandler_Success(t *testing.T) {
	// Arrange: create mock paired devices
	devices := []*devices.PairedDevice{
		{
			ID:             "device-001",
			Name:           "Living Room Hub",
			Token:          "token-abc-123",
			TokenExpiresAt: time.Now().Add(24 * time.Hour),
			PairedAt:       time.Now().Add(-30 * 24 * time.Hour),
			LastSeenAt:     time.Now().Add(-1 * time.Hour),
			IPAddress:      "192.168.1.100",
			UserAgent:      "MoonHub-iOS/1.0.0",
		},
		{
			ID:             "device-002",
			Name:           "Bedroom Display",
			Token:          "token-def-456",
			TokenExpiresAt: time.Now().Add(48 * time.Hour),
			PairedAt:       time.Now().Add(-7 * 24 * time.Hour),
			LastSeenAt:     time.Now().Add(-2 * time.Hour),
			IPAddress:      "192.168.1.101",
			UserAgent:      "MoonHub-Android/1.0.0",
		},
	}

	handler := newTestDevicesHandler(devices)

	// Act: create request from LAN IP
	req := httptest.NewRequest("GET", "/api/devices", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	handler.handleListDevices(w, req)

	// Assert: check response
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", response["success"])
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", response["data"])
	}

	if len(data) != 2 {
		t.Errorf("expected 2 devices, got %d", len(data))
	}

	// Verify first device
	firstDevice, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected device to be an object, got %T", data[0])
	}

	if firstDevice["id"] != "device-001" {
		t.Errorf("expected device id device-001, got %v", firstDevice["id"])
	}

	if firstDevice["name"] != "Living Room Hub" {
		t.Errorf("expected device name Living Room Hub, got %v", firstDevice["name"])
	}
}

func TestDevicesHandler_EmptyList(t *testing.T) {
	// Arrange: empty device list
	handler := newTestDevicesHandler([]*devices.PairedDevice{})

	// Act: create request from LAN IP
	req := httptest.NewRequest("GET", "/api/devices", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	handler.handleListDevices(w, req)

	// Assert: check response
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", response["success"])
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", response["data"])
	}

	if len(data) != 0 {
		t.Errorf("expected 0 devices, got %d", len(data))
	}
}

func TestDevicesHandler_LANRestriction(t *testing.T) {
	// Arrange: create handler with mock devices
	devices := []*devices.PairedDevice{
		{
			ID:             "device-001",
			Name:           "Test Device",
			Token:          "token-abc-123",
			TokenExpiresAt: time.Now().Add(24 * time.Hour),
			PairedAt:       time.Now(),
			LastSeenAt:     time.Now(),
		},
	}
	handler := newTestDevicesHandler(devices)

	// Act: create request from non-LAN IP (public IP)
	req := httptest.NewRequest("GET", "/api/devices", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	w := httptest.NewRecorder()

	handler.handleListDevices(w, req)

	// Assert: check response is forbidden
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", res.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	success, ok := response["success"].(bool)
	if ok && success {
		t.Error("expected success=false or missing, got true")
	}

	if response["error"] == nil {
		t.Error("expected error message in response")
	}
}

func TestDevicesHandler_LoopbackAllowed(t *testing.T) {
	// Arrange: create handler with mock devices
	devices := []*devices.PairedDevice{
		{
			ID:             "device-001",
			Name:           "Test Device",
			Token:          "token-abc-123",
			TokenExpiresAt: time.Now().Add(24 * time.Hour),
			PairedAt:       time.Now(),
			LastSeenAt:     time.Now(),
		},
	}
	handler := newTestDevicesHandler(devices)

	// Act: create request from loopback IP
	req := httptest.NewRequest("GET", "/api/devices", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.handleListDevices(w, req)

	// Assert: check response is OK (loopback is allowed)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for loopback, got %d", res.StatusCode)
	}
}

func TestNewDevicesHandler(t *testing.T) {
	// Test constructor
	store := &mockDeviceStore{devices: []*devices.PairedDevice{}}
	handler := &DevicesHandler{
		deviceStore: store,
	}

	// Assert the deviceStore field was set correctly
	if handler.deviceStore != store {
		t.Error("deviceStore field not set correctly")
	}
}

func TestDevicesHandler_RegisterRoutes(t *testing.T) {
	// Test route registration
	handler := &DevicesHandler{
		deviceStore: &mockDeviceStore{devices: []*devices.PairedDevice{}},
	}
	mux := http.NewServeMux()

	handler.RegisterRoutes(mux)

	// Verify route is registered by making a request
	req := httptest.NewRequest("GET", "/api/devices", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should not panic; response may be 403 due to LAN restriction
	res := w.Result()
	res.Body.Close()

	// We expect either 403 (LAN restriction) or 200 (if test environment allows loopback)
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusForbidden {
		t.Errorf("unexpected status code: %d", res.StatusCode)
	}
}

func TestDevicesHandler_SingleDevice(t *testing.T) {
	// Arrange: single device
	devices := []*devices.PairedDevice{
		{
			ID:             "device-single",
			Name:           "Single Device",
			Token:          "token-single-123",
			TokenExpiresAt: time.Now().Add(24 * time.Hour),
			PairedAt:       time.Now(),
			LastSeenAt:     time.Now(),
			IPAddress:      "192.168.1.200",
		},
	}

	handler := newTestDevicesHandler(devices)

	// Act: create request from LAN IP
	req := httptest.NewRequest("GET", "/api/devices", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	handler.handleListDevices(w, req)

	// Assert: check response
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", response["success"])
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", response["data"])
	}

	if len(data) != 1 {
		t.Errorf("expected 1 device, got %d", len(data))
	}

	// Verify device fields
	device, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected device to be an object, got %T", data[0])
	}

	if device["id"] != "device-single" {
		t.Errorf("expected device id device-single, got %v", device["id"])
	}

	if device["name"] != "Single Device" {
		t.Errorf("expected device name Single Device, got %v", device["name"])
	}
}

func TestDevicesHandler_IPv6LAN(t *testing.T) {
	// Arrange: create handler with mock devices
	devices := []*devices.PairedDevice{
		{
			ID:             "device-001",
			Name:           "Test Device",
			Token:          "token-abc-123",
			TokenExpiresAt: time.Now().Add(24 * time.Hour),
			PairedAt:       time.Now(),
			LastSeenAt:     time.Now(),
		},
	}
	handler := newTestDevicesHandler(devices)

	// Act: create request from IPv6 private address
	req := httptest.NewRequest("GET", "/api/devices", nil)
	req.RemoteAddr = "[fe80::1]:12345"
	w := httptest.NewRecorder()

	handler.handleListDevices(w, req)

	// Assert: check response is OK (IPv6 link-local is allowed)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for IPv6 link-local, got %d", res.StatusCode)
	}
}

func TestDevicesHandler_PrivateNetworkAllowed(t *testing.T) {
	// Test different private network ranges
	testCases := []struct {
		name        string
		remoteAddr  string
		expectAllow bool
	}{
		{"10.0.0.0", "10.0.0.1:12345", true},
		{"172.16.0.0", "172.16.0.1:12345", true},
		{"192.168.0.0", "192.168.0.1:12345", true},
		{"Public IP", "1.1.1.1:12345", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			devices := []*devices.PairedDevice{
				{
					ID:             "device-001",
					Name:           "Test Device",
					Token:          "token-abc-123",
					TokenExpiresAt: time.Now().Add(24 * time.Hour),
					PairedAt:       time.Now(),
					LastSeenAt:     time.Now(),
				},
			}
			handler := newTestDevicesHandler(devices)

			req := httptest.NewRequest("GET", "/api/devices", nil)
			req.RemoteAddr = tc.remoteAddr
			w := httptest.NewRecorder()

			handler.handleListDevices(w, req)

			res := w.Result()
			defer res.Body.Close()

			if tc.expectAllow {
				if res.StatusCode != http.StatusOK {
					t.Errorf("expected status 200 for %s, got %d", tc.name, res.StatusCode)
				}
			} else {
				if res.StatusCode != http.StatusForbidden {
					t.Errorf("expected status 403 for %s, got %d", tc.name, res.StatusCode)
				}
			}
		})
	}
}
