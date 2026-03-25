# MoonHub 局域网设备发现实现文档

> 状态: ✅ 已完成
> 版本: 1.0
> 更新日期: 2026-03-25

## 概述

本文档描述 MoonHub 局域网设备发现功能的实现，包括 mDNS 服务广播和客户端发现机制。

## 相关 Phase

- **Phase 1**: mDNS 服务发现 + 配对认证

## 模块结构

```
pkg/mdns/
├── server.go      # mDNS 服务端 - 广播设备信息
├── client.go      # mDNS 客户端 - 发现局域网设备
├── types.go       # 类型定义
├── server_test.go # 服务端测试
└── client_test.go # 客户端测试
```

## 1. mDNS 服务端 (`pkg/mdns/server.go`)

### 1.1 服务配置

```go
type ServerConfig struct {
    DeviceID   string        // 稳定设备 ID（来自 .moonhub_lan_device_id）
    Name       string        // 设备名称
    Version    string        // 软件版本
    Port       int           // API 端口（默认 18800）
    Iface      *net.Interface // 绑定网卡（可选）
}
```

### 1.2 服务广播

- **服务名称**: `_moonhub._tcp.local.`
- **实例名格式**: `MoonHub-{DeviceID} ({hostname})`

### 1.3 TXT 记录

```
id=<stable-device-uuid>
name=<device-name>
version=<version>
port=<api-port>
```

### 1.4 使用示例

```go
import "github.com/yourorg/moonhub/pkg/mdns"

// 创建服务端
server := mdns.NewServer(mdns.ServerConfig{
    DeviceID: "abc123",
    Name:     "My MoonHub",
    Version:  "1.0.0",
    Port:     18800,
})

// 启动广播
if err := server.Start(ctx); err != nil {
    log.Fatal(err)
}
defer server.Stop()
```

## 2. mDNS 客户端 (`pkg/mdns/client.go`)

### 2.1 发现接口

```go
type Client struct {
    timeout time.Duration
}

func (c *Client) Discover(ctx context.Context) ([]DeviceInfo, error)
```

### 2.2 设备信息

```go
type DeviceInfo struct {
    ID      string
    Name    string
    Version string
    Addr    net.IP
    Port    int
}
```

## 3. 稳定设备 ID

### 3.1 存储位置

- **文件**: `.moonhub_lan_device_id`
- **目录**: 与 `config.json` 同目录
- **格式**: UUID 字符串（如 `550e8400-e29b-41d4-a716-446655440000`）

### 3.2 生成逻辑

```go
// web/backend/utils/lan_device_id.go
func GetOrCreateDeviceID(configDir string) (string, error)
```

首次启动时生成并持久化，保证跨重启、跨主机名变更仍可被客户端识别。

## 4. API 端点

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/api/ping` | GET | 设备在线检测 | 无 |
| `/api/system/info` | GET | 设备详细状态 | 无 |

### 4.1 `/api/ping` 响应

```json
{
  "version": "1.0.0",
  "name": "MoonHub Device"
}
```

### 4.2 `/api/system/info` 响应

```json
{
  "version": "1.0.0",
  "name": "MoonHub Device",
  "device_id": "abc123",
  "uptime": 3600,
  "memory_usage": 8388608,
  "go_version": "go1.21.0"
}
```

## 5. 访问控制

所有发现和认证相关 API 仅允许局域网来源访问。

详见: `web/backend/api/lan_client.go`

```go
func IsLANRequest(r *http.Request) bool
```

允许的 IP 范围:
- `127.0.0.0/8` (回环)
- `10.0.0.0/8` (Class A 私网)
- `172.16.0.0/12` (Class B 私网)
- `192.168.0.0/16` (Class C 私网)
- `169.254.0.0/16` (链路本地)
- `fc00::/7` (IPv6 私网)
- `fe80::/10` (IPv6 链路本地)

## 6. 测试验证

```bash
# 运行 mDNS 测试
go test ./pkg/mdns/... -v

# 手动测试 - 检查设备广播
dns-sd -B _moonhub._tcp local.

# 手动测试 - 检查 API
curl http://127.0.0.1:18800/api/ping
curl http://127.0.0.1:18800/api/system/info
```

## 7. 依赖

| 包 | 版本 | 用途 |
|---|---|---|
| `github.com/hashicorp/mdns` | latest | mDNS 服务实现 |

## 8. 与 PWA 端交互

PWA 端通过 HTTP API 发现设备，而非直接使用 mDNS（浏览器限制）。

**发现流程**:
1. PWA 获取本机 IP（WebRTC）
2. 扫描同网段 IP 的常用端口
3. 对每个地址调用 `/api/ping`
4. 收集响应设备列表

详见: `MoonHub-PWA/src/services/discovery.ts`

## 9. 已知限制

1. **mDNS 在浏览器不可用**: PWA 需要通过 HTTP 扫描
2. **多网卡环境**: 默认绑定所有网卡，可通过 `Iface` 配置
3. **IPv6 支持**: 完整支持，但 PWA 扫描主要使用 IPv4

## 10. 未来改进

- [ ] 支持 mDNS 元数据更新（如状态变化）
- [ ] 优化 PWA 扫描性能（并行度控制）
- [ ] 支持 DNS-SD 服务子类型
