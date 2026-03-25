**Repository Documentation Index**: [docs/README.md](../../docs/README.md)

# pkg/devices - Device Management & Pairing Authentication

LAN device pairing and authentication module for managing paired devices, authorization codes, and tokens.

## Overview

This module implements MoonHub device pairing authentication:
- **Pairing Management**: Generate/validate authorization codes
- **Token Management**: Generate/verify authentication tokens
- **Device Storage**: Persist paired device information

## Core Components

### PairingManager

Manages the pairing flow with authorization codes:

```go
import (
    "github.com/yourorg/moonhub/pkg/devices"
)

// Create device store
store := devices.NewDeviceStore("/path/to/config/dir")

// Create pairing manager
pairingMgr := devices.NewPairingManager(store)

// Generate authorization code
code, err := pairingMgr.GenerateCodeOnce()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Pairing code: %s\n", code)
```

### DeviceStore

Manages paired device persistence:

```go
// Add a paired device
err := store.AddDevice(&devices.PairedDevice{
    ID:        "client-device-id",
    Name:      "My iPhone",
    IPAddress: "192.168.1.100",
})

// Validate token
device, err := store.ValidateToken(tokenFromRequest)
if err != nil {
    return unauthorized()
}

// Update last seen
store.UpdateLastSeen(device.ID, clientIP)
```

### Token Management

Tokens provide ongoing authentication:

```go
// Token is generated during pairing
result, err := pairingMgr.ValidateCode("XM8888", deviceInfo)
if result.Success {
    fmt.Printf("Token: %s\n", result.Token)
}

// Validate token on requests
device, err := store.ValidateToken(token)
```

## Configuration

See [docs/CONFIG.md](./docs/CONFIG.md) for detailed configuration options.

Quick reference:

| Component | Setting | Default |
|-----------|---------|---------|
| Auth Code | Format | 2 letters + 4 digits |
| Auth Code | Validity | 5 minutes |
| Auth Code | Usage | One-time |
| Token | Length | 32 bytes |
| Token | Validity | 30 days |
| Storage | File | `paired_devices.json` |

## Type Definitions

### PairingManager

```go
type PairingManager struct {
    // Internal state for current auth code and device store
}

// Generate auth code (returns existing if valid)
func (pm *PairingManager) GenerateCodeOnce() (string, error)

// Get current auth code
func (pm *PairingManager) GetCurrentCode() string

// Validate code and complete pairing
func (pm *PairingManager) ValidateCode(code string, deviceInfo *PairedDevice) (*PairingResult, error)

// Regenerate auth code
func (pm *PairingManager) RegenerateCode() (string, error)

// Check if any devices are paired
func (pm *PairingManager) IsPaired() bool
```

### DeviceStore

```go
type DeviceStore struct {
    // Manages paired devices
}

// Add device
func (s *DeviceStore) AddDevice(device *PairedDevice) error

// Get device by ID
func (s *DeviceStore) GetDevice(id string) (*PairedDevice, error)

// Get all paired devices
func (s *DeviceStore) GetAllDevices() ([]*PairedDevice, error)

// Remove device
func (s *DeviceStore) RemoveDevice(id string) error

// Validate token
func (s *DeviceStore) ValidateToken(token string) (*PairedDevice, error)

// Update last access time
func (s *DeviceStore) UpdateLastSeen(id string, ip string) error
```

### PairedDevice

```go
type PairedDevice struct {
    ID              string    `json:"id"`
    Name            string    `json:"name"`
    Token           string    `json:"token"`
    TokenExpiresAt  time.Time `json:"token_expires_at"`
    PairedAt        time.Time `json:"paired_at"`
    LastSeenAt      time.Time `json:"last_seen_at"`
    IPAddress       string    `json:"ip_address"`
    UserAgent       string    `json:"user_agent,omitempty"`
}
```

## Usage Examples

### Server-Side Pairing Flow

```go
// 1. Generate and display auth code
code, _ := pairingMgr.GenerateCodeOnce()
fmt.Printf("Enter code in PWA: %s\n", code)

// 2. Client calls pair endpoint with code
func handlePair(w http.ResponseWriter, r *http.Request) {
    var req PairRequest
    json.NewDecoder(r.Body).Decode(&req)

    deviceInfo := &devices.PairedDevice{
        ID:        uuid.New().String(),
        Name:      req.DeviceName,
        IPAddress: r.RemoteAddr,
    }

    result, err := pairingMgr.ValidateCode(req.Code, deviceInfo)
    if err != nil || !result.Success {
        http.Error(w, "Invalid code", 401)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{
        "token": result.Token,
    })
}
```

### Token Validation Middleware

```go
func AuthMiddleware(store *devices.DeviceStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                http.Error(w, "Unauthorized", 401)
                return
            }

            device, err := store.ValidateToken(token)
            if err != nil {
                http.Error(w, "Invalid token", 401)
                return
            }

            // Update last seen
            store.UpdateLastSeen(device.ID, r.RemoteAddr)

            // Add device to context
            ctx := context.WithValue(r.Context(), "device", device)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Device Management

```go
// List all paired devices
devices, _ := store.GetAllDevices()
for _, d := range devices {
    fmt.Printf("%s (%s) - Last seen: %s\n", d.Name, d.IPAddress, d.LastSeenAt)
}

// Remove a device
store.RemoveDevice("device-id")
```

## Testing

```bash
# Run all tests
go test ./pkg/devices/... -v

# Run specific tests
go test ./pkg/devices/... -v -run TestPairingManager
go test ./pkg/devices/... -v -run TestDeviceStore
go test ./pkg/devices/... -v -run TestToken
```

## Related Documentation

- [Pairing Implementation Status](../../docs/implementation/lan-pairing-status.md)
- [Configuration Options](./docs/CONFIG.md)
