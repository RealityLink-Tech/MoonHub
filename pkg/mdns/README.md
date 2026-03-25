**Repository Documentation Index**: [docs/README.md](../../docs/README.md)

# pkg/mdns - mDNS Service Discovery

mDNS (Multicast DNS) based LAN device discovery module for broadcasting and discovering MoonHub devices.

## Overview

This module implements automatic discovery of MoonHub devices on the local network:
- **Server**: Broadcasts `_moonhub._tcp.local.` service
- **Client**: Discovers all MoonHub devices on the LAN

## Core Components

### Server

The mDNS server broadcasts MoonHub service presence:

```go
import (
    "context"
    "github.com/yourorg/moonhub/pkg/mdns"
)

// Create server
server := mdns.NewServer(mdns.ServerConfig{
    DeviceID: "abc123",
    Name:     "My MoonHub",
    Version:  "1.0.0",
    Port:     18800,
})

// Start broadcasting
if err := server.Start(context.Background()); err != nil {
    log.Fatal(err)
}
defer server.Stop()
```

### Client

The mDNS client discovers MoonHub devices:

```go
import (
    "context"
    "time"
    "github.com/yourorg/moonhub/pkg/mdns"
)

// Create client with 5 second timeout
client := mdns.NewClient(5 * time.Second)

// Discover devices
devices, err := client.Discover(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, device := range devices {
    fmt.Printf("Found: %s at %s:%d\n", device.Name, device.Addr, device.Port)
}
```

### DeviceInfo

Device information structure returned by discovery:

```go
type DeviceInfo struct {
    ID      string    // Device ID
    Name    string    // Device name
    Version string    // Software version
    Addr    net.IP    // IP address
    Port    int       // API port
}
```

## Configuration

See [docs/CONFIG.md](./docs/CONFIG.md) for detailed configuration options.

Quick reference:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `DeviceID` | string | required | Stable device identifier |
| `Name` | string | required | Human-readable device name |
| `Version` | string | required | Software version |
| `Port` | int | 18800 | API port |
| `Iface` | *net.Interface | nil | Network interface to bind |

## Service Constants

```go
const (
    ServiceName    = "_moonhub._tcp"
    ServiceDomain  = "local."
    DefaultPort    = 18800
)
```

## TXT Records

Service TXT record keys:

| Key | Description |
|-----|-------------|
| `id` | Stable device ID |
| `name` | Device name |
| `version` | Software version |
| `port` | API port |

## Testing

```bash
# Run tests
go test ./pkg/mdns/... -v

# Manual test - check service broadcast
dns-sd -B _moonhub._tcp local.

# Manual test - resolve service
dns-sd -L "My MoonHub" _moonhub._tcp local.
```

## Dependencies

- `github.com/hashicorp/mdns` - mDNS implementation

## Related Documentation

- [LAN Discovery Status](../../docs/implementation/lan-discovery-status.md)
- [Configuration Options](./docs/CONFIG.md)
