# MoonHub 局域网配对认证实现文档

> 状态: ✅ 已完成
> 版本: 1.0
> 更新日期: 2026-03-25

## 概述

本文档描述 MoonHub 局域网设备配对认证功能的实现，包括授权码管理和 Token 认证机制。

## 相关 Phase

- **Phase 1**: mDNS 服务发现 + 配对认证

## 模块结构

```
pkg/devices/
├── store.go       # 设备存储 - 管理 paired_devices.json
├── pairing.go     # 配对管理 - 授权码生成/验证
├── token.go       # Token 管理 - 生成/校验
├── types.go       # 类型定义
└── pairing_test.go # 配对测试

web/backend/api/
├── auth_pair.go   # 配对认证 API 端点
└── lan.go         # 局域网访问控制
```

## 1. 配对管理 (`pkg/devices/pairing.go`)

### 1.1 PairingManager

```go
type PairingManager struct {
    mu          sync.RWMutex
    current     *PairingCode
    usedCodes   map[string]*PairingCode
    deviceStore *DeviceStore
}
```

### 1.2 授权码规格

| 属性 | 值 |
|-----|---|
| 格式 | 2 字母 + 4 数字（如 `XM8888`） |
| 有效期 | 5 分钟 |
| 使用次数 | 一次性（使用后失效） |
| 大小写 | 大写 |

### 1.3 主要方法

```go
// 生成授权码（如已存在且未过期则返回现有码）
func (pm *PairingManager) GenerateCodeOnce() (string, error)

// 获取当前授权码
func (pm *PairingManager) GetCurrentCode() string

// 验证授权码并配对
func (pm *PairingManager) ValidateCode(code string, deviceInfo *PairedDevice) (*PairingResult, error)

// 重新生成授权码
func (pm *PairingManager) RegenerateCode() (string, error)

// 检查是否有已配对设备
func (pm *PairingManager) IsPaired() bool
```

## 2. Token 管理 (`pkg/devices/token.go`)

### 2.1 Token 规格

| 属性 | 值 |
|-----|---|
| 长度 | 32 字节随机 |
| 有效期 | 30 天 |
| 存储格式 | 64 字符小写 hex |
| 传输方式 | Bearer Token |

### 2.2 Token 生成

```go
func GenerateToken() (token string, expiresAt time.Time, err error)
```

### 2.3 兼容性

为兼容已配对的旧设备，校验接口仍接受 **旧版 base64url Token**，新配对仅发放 hex 格式。

## 3. 设备存储 (`pkg/devices/store.go`)

### 3.1 DeviceStore

```go
type DeviceStore struct {
    path     string  // paired_devices.json 绝对路径
    devices  map[string]*PairedDevice
    mu       sync.RWMutex
}
```

### 3.2 存储位置

- **文件**: `paired_devices.json`
- **目录**: 与 `config.json` 同目录
- **默认**: `~/.moonhub/paired_devices.json`

### 3.3 PairedDevice 结构

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

## 4. API 端点

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/api/auth/status` | GET | 获取当前授权码状态 | 无（LAN Only） |
| `/api/auth/pair` | POST | 使用授权码配对 | 无（LAN Only） |
| `/api/auth/verify` | POST | 验证 Token 有效性 | Bearer Token |

### 4.1 GET `/api/auth/status`

**响应**:
```json
{
  "success": true,
  "data": {
    "code": "XM8888",
    "expiresAt": "2026-03-25T12:05:00Z",
    "isPaired": false
  }
}
```

### 4.2 POST `/api/auth/pair`

**请求**:
```json
{
  "code": "XM8888",
  "deviceId": "pwa-device-uuid",
  "deviceName": "My iPhone"
}
```

> 字段别名: `device_id` / `device_name`（snake_case 也接受）
> `deviceId` / `deviceName` 可省略，由服务端生成默认值

**成功响应**:
```json
{
  "success": true,
  "data": {
    "token": "64位hex字符串",
    "tokenExpiresAt": "2026-04-25T00:00:00Z",
    "device": {
      "id": "pwa-device-uuid",
      "name": "My iPhone",
      "token": "64位hex字符串",
      "tokenExpiresAt": "2026-04-25T00:00:00Z",
      "pairedAt": "2026-03-25T12:00:00Z",
      "lastSeenAt": "2026-03-25T12:00:00Z",
      "ipAddress": "192.168.1.100"
    }
  }
}
```

**失败响应**:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CODE",
    "message": "invalid pairing code"
  }
}
```

### 4.3 POST `/api/auth/verify`

**传 Token 方式**（按优先级）:

1. **推荐**: 请求头 `Authorization: Bearer <token>`
2. **可选**: JSON 体 `{"token":"<token>"}`

**成功响应**:
```json
{
  "success": true,
  "data": {
    "valid": true,
    "deviceId": "pwa-device-uuid",
    "expiresAt": "2026-04-25T00:00:00Z"
  }
}
```

## 5. 认证流程

```
┌─────────────┐          ┌─────────────┐
│    PWA      │          │  MoonHub    │
└──────┬──────┘          └──────┬──────┘
       │                        │
       │  1. GET /api/auth/status
       │ ──────────────────────>│
       │                        │
       │  2. 返回授权码          │
       │ <──────────────────────│
       │                        │
       │  3. 用户输入授权码       │
       │                        │
       │  4. POST /api/auth/pair
       │ ──────────────────────>│
       │                        │
       │  5. 返回 Token          │
       │ <──────────────────────│
       │                        │
       │  6. POST /api/auth/verify (Bearer Token)
       │ ──────────────────────>│
       │                        │
       │  7. 验证成功            │
       │ <──────────────────────│
       │                        │
```

## 6. 访问控制

所有认证 API 仅允许局域网来源访问（`web/backend/api/lan.go`）。

```go
func IsLANRequest(r *http.Request) bool
```

## 7. 测试验证

```bash
# 运行配对测试
go test ./pkg/devices/... -v

# 手动测试 - 获取授权码
curl http://127.0.0.1:18800/api/auth/status

# 手动测试 - 配对
curl -X POST http://127.0.0.1:18800/api/auth/pair \
  -H "Content-Type: application/json" \
  -d '{"code":"XM8888","deviceId":"test-device","deviceName":"Test Device"}'

# 手动测试 - 验证 Token（Bearer 方式）
curl -X POST http://127.0.0.1:18800/api/auth/verify \
  -H "Authorization: Bearer <your-token>"

# 手动测试 - 验证 Token（JSON 方式）
curl -X POST http://127.0.0.1:18800/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"token":"<your-token>"}'
```

## 8. 安全考虑

### 8.1 授权码安全

- **有效期短**: 5 分钟窗口减少撞码风险
- **一次性使用**: 使用后立即失效
- **局域网限制**: 仅 LAN 内可访问

### 8.2 Token 安全

- **足够长度**: 32 字节 = 256 位熵
- **有效期限制**: 30 天自动过期
- **传输安全**: 建议配合 HTTPS（局域网内 HTTP 可接受）

### 8.3 已知风险

- 授权码可能被窥屏（用户需保护屏幕）
- Token 存储在客户端 IndexedDB（需设备本身安全）

## 9. 与 PWA 端交互

PWA 端配对流程详见: `MoonHub-PWA/docs/lan-pairing.md`

关键代码:
- `MoonHub-PWA/src/hooks/useDevice.ts` - 配对逻辑
- `MoonHub-PWA/src/services/device.ts` - API 调用
- `MoonHub-PWA/src/stores/device.ts` - 状态管理

## 10. 未来改进

- [ ] 支持 Token 撤销
- [ ] 支持多设备管理 UI
- [ ] 支持 Ed25519 签名认证（Phase 3）
- [ ] 支持 Token 刷新机制
