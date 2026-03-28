package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// TestDiscoveryAndAuthRoutesRegistered verifies that discovery and auth routes are properly registered.
func TestDiscoveryAndAuthRoutesRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	deviceStore, err := devices.NewDeviceStore(tmpDir)
	if err != nil {
		t.Fatalf("NewDeviceStore() error = %v", err)
	}
	pairingManager := devices.NewPairingManager(deviceStore)

	h := NewHandler(configPath, deviceStore, pairingManager)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Test discovery routes: GET /api/ping
	testCases := []struct {
		method       string
		path         string
		expectStatus int
		description  string
	}{
		{
			method:       "GET",
			path:         "/api/ping",
			expectStatus: http.StatusOK,
			description:  "Discovery ping endpoint should be registered",
		},
		{
			method:       "GET",
			path:         "/api/system/info",
			expectStatus: http.StatusOK,
			description:  "Discovery system info endpoint should be registered",
		},
		{
			method:       "GET",
			path:         "/api/auth/status",
			expectStatus: http.StatusOK,
			description:  "Auth status endpoint should be registered",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			mux.ServeHTTP(rec, req)

			// We expect either OK or a different success/error status
			// The important thing is that the route is registered (not 404)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s %s returned 404 (route not registered)", tc.method, tc.path)
			}
		})
	}
}

// TestDiscoveryHandlerNotNil verifies that the discovery handler is initialized.
func TestDiscoveryHandlerNotNil(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	deviceStore, err := devices.NewDeviceStore(tmpDir)
	if err != nil {
		t.Fatalf("NewDeviceStore() error = %v", err)
	}
	pairingManager := devices.NewPairingManager(deviceStore)

	h := NewHandler(configPath, deviceStore, pairingManager)

	if h.discovery == nil {
		t.Fatal("discovery handler should not be nil")
	}
	if h.auth == nil {
		t.Fatal("auth handler should not be nil")
	}
}
