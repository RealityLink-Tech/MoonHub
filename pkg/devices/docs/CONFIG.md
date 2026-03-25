**Repository Documentation Index**: [docs/README.md](../../../docs/README.md)

# Device Pairing Configuration

This document describes the configuration options for device pairing and token management.

## Overview

The device pairing system uses authorization codes for initial pairing and tokens for ongoing authentication.

## Authorization Code Configuration

### Format

| Property | Value | Description |
|----------|-------|-------------|
| Format | 2 letters + 4 digits | Human-readable code |
| Example | `XM8888`, `AB1234` | Easy to enter |
| Case | Uppercase | Normalized on input |

The format is designed to be:
- Easy to read and enter
- Short enough to remember
- Long enough for security

### Validity

| Property | Value |
|----------|-------|
| Lifetime | 5 minutes |
| Regeneration | On demand |
| Storage | In-memory |

Code lifecycle:
1. Generated on first request
2. Valid for 5 minutes
3. Invalidated after successful use
4. Can be regenerated manually

### Usage Rules

| Rule | Description |
|------|-------------|
| One-time | Code invalidated after successful pairing |
| Single active | Only one code valid at a time |
| Regeneratable | New code can be generated anytime |

## Token Configuration

### Format

| Property | Value | Description |
|----------|-------|-------------|
| Length | 32 bytes | Cryptographically random |
| Encoding | Hex (lowercase) | 64 character string |
| Example | `a1b2c3d4...` | 64 hex characters |

### Validity

| Property | Value |
|----------|-------|
| Lifetime | 30 days |
| Renewal | On pairing |
| Storage | JSON file |

Token lifecycle:
1. Generated on successful pairing
2. Stored on client device
3. Sent with each request
4. Validated on server
5. Expires after 30 days

### Transmission

| Method | Header |
|--------|--------|
| Bearer Token | `Authorization: Bearer <token>` |

## Storage Configuration

### File Location

| Property | Value |
|----------|-------|
| Filename | `paired_devices.json` |
| Directory | Same as `config.json` |
| Default | `~/.moonhub/paired_devices.json` |

### File Format

```json
{
  "devices": [
    {
      "id": "device-uuid",
      "name": "My iPhone",
      "token": "64-char-hex-string",
      "token_expires_at": "2024-02-20T00:00:00Z",
      "paired_at": "2024-01-20T10:30:00Z",
      "last_seen_at": "2024-01-25T15:45:00Z",
      "ip_address": "192.168.1.100",
      "user_agent": "MoonHub-PWA/1.0"
    }
  ]
}
```

### Directory Permissions

| Requirement | Value |
|-------------|-------|
| Directory | Read/Write |
| File | Read/Write |
| Mode | 0600 (owner only) |

## Access Control

### Pairing Policy

| Setting | Default | Description |
|---------|---------|-------------|
| Require code | Yes | Must enter valid code |
| Single code | Yes | One code at a time |
| Code timeout | 5 min | Code expiration |

### Token Policy

| Setting | Default | Description |
|---------|---------|-------------|
| Multiple devices | Yes | Multiple devices can pair |
| Token refresh | No | Tokens don't auto-refresh |
| Expiration | 30 days | Token validity period |

## Usage Examples

### Custom Configuration

```go
// Create store with custom path
store := devices.NewDeviceStore("/custom/config/path")

// Create pairing manager
pairingMgr := devices.NewPairingManager(store)
```

### Check Pairing Status

```go
// Check if any devices are paired
if pairingMgr.IsPaired() {
    fmt.Println("Devices are paired")
} else {
    code, _ := pairingMgr.GenerateCodeOnce()
    fmt.Printf("Pair with code: %s\n", code)
}
```

### Regenerate Code

```go
// Force regenerate code (invalidates old one)
newCode, err := pairingMgr.RegenerateCode()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("New code: %s\n", newCode)
```

## Security Considerations

### Authorization Codes

- Short-lived (5 minutes)
- Single use only
- Displayed locally only
- Not transmitted over network until pairing

### Tokens

- Cryptographically random
- Long enough to prevent brute force
- Transmitted over HTTPS
- Stored securely on both ends

### Storage

- File permissions restricted to owner
- Tokens hashed for comparison
- Automatic cleanup of expired tokens

## Troubleshooting

### Code Expired

```
Error: authorization code expired
Solution: Generate new code and retry
```

### Code Already Used

```
Error: authorization code already used
Solution: Generate new code and retry
```

### Token Invalid

```
Error: invalid token
Solution: Re-pair device with new authorization code
```

### Token Expired

```
Error: token expired
Solution: Re-pair device with new authorization code
```

## Notes

- Authorization codes are case-insensitive
- Multiple devices can be paired to one MoonHub
- Tokens are unique per device
- Storage file is created on first pairing
- Expired tokens are validated but rejected
