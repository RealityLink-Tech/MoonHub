package api

import (
	"net/http"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// setupAuthenticatedDevice creates a paired device in deviceStore with a valid
// token and returns that token. Tests should set "Authorization: Bearer <token>"
// on requests to protected endpoints.
func setupAuthenticatedDevice(t *testing.T, deviceStore *devices.DeviceStore) string {
	t.Helper()
	token, expiresAt, err := devices.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	device := &devices.PairedDevice{
		ID:             "test-device",
		Name:           "Test Device",
		Token:          token,
		TokenExpiresAt: expiresAt,
	}
	if err := deviceStore.AddDevice(device); err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}
	return token
}

// withBearerToken sets the Authorization header on the request to the given token.
func withBearerToken(r *http.Request, token string) *http.Request {
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}
