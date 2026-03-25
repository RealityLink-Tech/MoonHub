**Repository Documentation Index**: [docs/README.md](../../../docs/README.md)

# mDNS Configuration

This document describes the configuration options for the mDNS service discovery module.

## Overview

The mDNS module supports various configuration options for service broadcasting and discovery.

## ServerConfig

Configure the mDNS server in code:

```go
type ServerConfig struct {
    DeviceID   string         // Stable device ID
    Name       string         // Device name
    Version    string         // Software version
    Port       int            // API port (default 18800)
    Iface      *net.Interface // Network interface to bind (optional)
}
```

## Configuration Options

### DeviceID

- **Type**: `string`
- **Required**: Yes
- **Description**: Stable, unique identifier for the device

The DeviceID should be:
- Persistent across restarts
- Unique within the network
- Typically a UUID or hash of hardware identifiers

Example:
```go
DeviceID: "moonhub-abc123def456"
```

### Name

- **Type**: `string`
- **Required**: Yes
- **Description**: Human-readable device name displayed during discovery

Example:
```go
Name: "Living Room MoonHub"
```

### Version

- **Type**: `string`
- **Required**: Yes
- **Description**: Software version for compatibility checking

Example:
```go
Version: "1.0.0"
```

### Port

- **Type**: `int`
- **Default**: `18800`
- **Description**: TCP port for the API server

The API port must:
- Match the actual HTTP server port
- Be accessible on the local network
- Not conflict with other services

Example:
```go
Port: 18800
```

### Iface

- **Type**: `*net.Interface`
- **Default**: `nil` (all interfaces)
- **Description**: Specific network interface to bind

Use this to:
- Bind to a specific network adapter
- Avoid broadcasting on unwanted interfaces
- Support multi-homed systems

Example:
```go
// Bind to specific interface
iface, _ := net.InterfaceByName("eth0")
Iface: iface
```

## TXT Record Format

The service broadcasts the following TXT records:

| Key | Type | Description |
|-----|------|-------------|
| `id` | string | Device ID (matches DeviceID) |
| `name` | string | Device name (matches Name) |
| `version` | string | Software version (matches Version) |

TXT records are used for:
- Device identification without connection
- Version compatibility checking
- Displaying device info in discovery UI

## Client Configuration

The mDNS client uses timeout-based discovery:

```go
type ClientConfig struct {
    Timeout     time.Duration // Discovery timeout
    ServiceName string        // Service to discover (default: _moonhub._tcp)
}
```

### Timeout

- **Type**: `time.Duration`
- **Default**: `5 * time.Second`
- **Description**: Maximum time to wait for discovery

Longer timeouts:
- Find more devices on larger networks
- Increase user wait time
- May be needed for slow networks

Example:
```go
client := mdns.NewClient(10 * time.Second)
```

## Network Requirements

### Ports

| Port | Protocol | Usage |
|------|----------|-------|
| 5353 | UDP | mDNS multicast |
| 18800 | TCP | API server |

### Multicast Address

- IPv4: `224.0.0.251`
- IPv6: `ff02::fb`

### Firewall Rules

Ensure the following are allowed:
- UDP port 5353 for mDNS
- TCP port 18800 for API access

## Usage Examples

### Basic Server Setup

```go
server := mdns.NewServer(mdns.ServerConfig{
    DeviceID: generateDeviceID(),
    Name:     "My MoonHub",
    Version:  "1.0.0",
    Port:     18800,
})
server.Start(context.Background())
```

### Multi-Interface Setup

```go
// Get WiFi interface
wifiIface, _ := net.InterfaceByName("wlan0")

server := mdns.NewServer(mdns.ServerConfig{
    DeviceID: deviceID,
    Name:     "My MoonHub",
    Version:  "1.0.0",
    Port:     18800,
    Iface:    wifiIface,
})
```

### Custom Discovery Timeout

```go
// 10 second timeout for larger networks
client := mdns.NewClient(10 * time.Second)
devices, err := client.Discover(context.Background())
```

## Troubleshooting

### Devices Not Found

1. Check firewall allows UDP 5353
2. Verify devices are on same subnet
3. Try increasing discovery timeout
4. Check mDNS service running with `dns-sd -B _moonhub._tcp local.`

### Service Not Broadcasting

1. Verify server started without error
2. Check port 18800 is available
3. Verify network interface is up
4. Check logs for mDNS errors

## Notes

- mDNS works only on local network (same subnet)
- Device discovery may take up to the configured timeout
- Service broadcast continues until server is stopped
- Multiple MoonHub instances on same network are supported
