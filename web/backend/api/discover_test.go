package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/mdns"
)

// mockMDNSClient is a mock implementation of the mDNS client for testing.
type mockMDNSClient struct {
	devices    []mdns.DeviceInfo
	err        error
	delay      time.Duration
	timeoutSim bool
}

func (m *mockMDNSClient) Discover(ctx context.Context) ([]mdns.DeviceInfo, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.timeoutSim {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return []mdns.DeviceInfo{}, nil
		}
	}

	return m.devices, m.err
}

// newTestDiscoverHandler creates a DiscoverHandler with a mock client.
func newTestDiscoverHandler(devices []mdns.DeviceInfo, err error) *DiscoverHandler {
	return &DiscoverHandler{
		mdnsClient: &mockMDNSClient{devices: devices, err: err},
	}
}

func TestDiscoverHandler_Success(t *testing.T) {
	// Arrange: create mock devices
	devices := []mdns.DeviceInfo{
		{
			ID:      "YSHU-2026-ABC",
			Name:    "MoonHub-LivingRoom",
			Version: "1.0.0",
			Addr:    net.ParseIP("192.168.1.100"),
			Port:    18800,
		},
		{
			ID:      "YSHU-2026-DEF",
			Name:    "MoonHub-Bedroom",
			Version: "1.0.0",
			Addr:    net.ParseIP("192.168.1.101"),
			Port:    18800,
		},
	}

	handler := newTestDiscoverHandler(devices, nil)

	// Act: create request from LAN IP
	req := httptest.NewRequest("GET", "/api/discover", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	handler.handleDiscover(w, req)

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

	if firstDevice["id"] != "YSHU-2026-ABC" {
		t.Errorf("expected device id YSHU-2026-ABC, got %v", firstDevice["id"])
	}

	if firstDevice["name"] != "MoonHub-LivingRoom" {
		t.Errorf("expected device name MoonHub-LivingRoom, got %v", firstDevice["name"])
	}
}

func TestDiscoverHandler_NoDevices(t *testing.T) {
	// Arrange: empty device list
	handler := newTestDiscoverHandler([]mdns.DeviceInfo{}, nil)

	// Act: create request from LAN IP
	req := httptest.NewRequest("GET", "/api/discover", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	handler.handleDiscover(w, req)

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

func TestDiscoverHandler_LANRestriction(t *testing.T) {
	// Arrange: create handler with mock devices
	devices := []mdns.DeviceInfo{
		{ID: "YSHU-2026-ABC", Name: "TestDevice", Addr: net.ParseIP("192.168.1.100"), Port: 18800},
	}
	handler := newTestDiscoverHandler(devices, nil)

	// Act: create request from non-LAN IP (public IP)
	req := httptest.NewRequest("GET", "/api/discover", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	w := httptest.NewRecorder()

	handler.handleDiscover(w, req)

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

func TestDiscoverHandler_LoopbackAllowed(t *testing.T) {
	// Arrange: create handler with mock devices
	devices := []mdns.DeviceInfo{
		{ID: "YSHU-2026-ABC", Name: "TestDevice", Addr: net.ParseIP("192.168.1.100"), Port: 18800},
	}
	handler := newTestDiscoverHandler(devices, nil)

	// Act: create request from loopback IP
	req := httptest.NewRequest("GET", "/api/discover", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.handleDiscover(w, req)

	// Assert: check response is OK (loopback is allowed)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for loopback, got %d", res.StatusCode)
	}
}

func TestDiscoverHandler_ContextTimeout(t *testing.T) {
	// Arrange: create mock client that simulates timeout behavior
	slowClient := &mockMDNSClient{
		devices:    []mdns.DeviceInfo{},
		timeoutSim: true,
	}

	handler := &DiscoverHandler{
		mdnsClient: slowClient,
	}

	// Act: create request with very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/api/discover", nil).WithContext(ctx)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	// Use a goroutine to avoid blocking test
	done := make(chan bool)
	go func() {
		handler.handleDiscover(w, req)
		done <- true
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(200 * time.Millisecond):
		t.Error("handler did not complete within expected time")
	}

	// Assert: check response
	res := w.Result()
	defer res.Body.Close()

	// Should succeed with empty result due to timeout
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
}

func TestNewDiscoverHandler(t *testing.T) {
	// Test constructor
	client := mdns.NewClient()
	handler := NewDiscoverHandler(client)

	if handler == nil {
		t.Fatal("NewDiscoverHandler returned nil")
	}

	if handler.mdnsClient != client {
		t.Error("mdnsClient field not set correctly")
	}
}

func TestDiscoverHandler_RegisterRoutes(t *testing.T) {
	// Test route registration
	handler := NewDiscoverHandler(mdns.NewClient())
	mux := http.NewServeMux()

	handler.RegisterRoutes(mux)

	// Verify route is registered by making a request
	// (This is a basic smoke test; more thorough testing would require internal inspection)
	req := httptest.NewRequest("GET", "/api/discover", nil)
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
