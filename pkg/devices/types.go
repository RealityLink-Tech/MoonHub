// Package devices provides device pairing and management for MoonHub.
package devices

import (
	"time"
)

// PairingCode represents an authorization code for device pairing.
type PairingCode struct {
	// Code is the 6-character pairing code (e.g., "XM8888").
	Code string `json:"code"`

	// CreatedAt is when the code was generated.
	CreatedAt time.Time `json:"createdAt"`

	// UsedAt is when the code was used (nil if unused).
	UsedAt *time.Time `json:"usedAt,omitempty"`

	// PairedWith is the device ID that used this code.
	PairedWith string `json:"pairedWith,omitempty"`
}

// PairedDevice represents a device that has been paired.
type PairedDevice struct {
	// ID is the unique identifier of the paired device.
	ID string `json:"id"`

	// Name is the friendly name of the device.
	Name string `json:"name"`

	// Token is the access token for this device.
	Token string `json:"token"`

	// TokenExpiresAt is when the token expires.
	TokenExpiresAt time.Time `json:"tokenExpiresAt"`

	// PairedAt is when the device was paired.
	PairedAt time.Time `json:"pairedAt"`

	// LastSeenAt is the last time the device was seen.
	LastSeenAt time.Time `json:"lastSeenAt"`

	// IPAddress is the last known IP address of the device.
	IPAddress string `json:"ipAddress,omitempty"`

	// UserAgent is the user agent string of the paired client.
	UserAgent string `json:"userAgent,omitempty"`
}

// DeviceStatus represents the current status of a device.
type DeviceStatus struct {
	// DeviceID is the unique identifier of this MoonHub device.
	DeviceID string `json:"deviceId"`

	// Name is the friendly name of the device.
	Name string `json:"name"`

	// Version is the software version.
	Version string `json:"version"`

	// Platform is the operating system platform.
	Platform string `json:"platform"`

	// Arch is the CPU architecture.
	Arch string `json:"arch"`

	// Uptime is how long the device has been running (in seconds).
	Uptime int64 `json:"uptime"`

	// Paired indicates if at least one device is paired.
	Paired bool `json:"paired"`

	// PairedCount is the number of paired devices.
	PairedCount int `json:"pairedCount"`
}

// PairingResult represents the result of a pairing operation.
type PairingResult struct {
	// Success indicates if pairing was successful.
	Success bool `json:"success"`

	// Token is the access token (only on success).
	Token string `json:"token,omitempty"`

	// TokenExpiresAt is when the token expires.
	TokenExpiresAt time.Time `json:"tokenExpiresAt,omitempty"`

	// Device is the paired device info.
	Device *PairedDevice `json:"device,omitempty"`

	// Error is the error message (only on failure).
	Error string `json:"error,omitempty"`
}

// TokenValidation represents the result of token validation.
type TokenValidation struct {
	// Valid indicates if the token is valid.
	Valid bool `json:"valid"`

	// DeviceID is the ID of the device (only if valid).
	DeviceID string `json:"deviceId,omitempty"`

	// ExpiresAt is when the token expires (only if valid).
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	// Error is the error message (only if invalid).
	Error string `json:"error,omitempty"`
}
