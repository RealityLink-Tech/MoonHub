# web/backend/api - HTTP API 端点

MoonHub Web 后端 HTTP API 实现。

## 概述

本目录包含所有 HTTP API 端点的实现：
- **discovery.go** - 设备发现 API
- **auth_pair.go** - 配对认证 API
- **lan.go** - 局域网访问控制
- **config.go** - 配置管理 API
- **chat.go** - 对话 API
- 等等...

## 文件结构

```
web/backend/api/
├── router.go           # 路由注册
├── discovery.go        # 设备发现 API
├── auth_pair.go        # 配对认证 API
├── lan.go              # 局域网访问控制
├── lan_client.go       # LAN 客户端工具
├── config.go           # 配置管理 API
├── models.go           # 模型管理 API
├── skills.go           # 技能管理 API
├── tools.go            # 工具管理 API
├── gateway.go          # 网关控制 API
├── events.go           # SSE 事件流
├── session.go          # 会话管理 API
├── oauth.go            # OAuth 认证
├── provisioning.go     # 设备配网 API
├── startup.go          # 启动配置 API
├── log.go              # 日志 API
└── channels.go         # 频道管理 API
```

## API 端点分类

### 设备发现（无需认证）

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/ping` | GET | 设备在线检测 | `discovery.go` |
| `/api/system/info` | GET | 设备详细状态 | `discovery.go` |

### 配对认证（无需认证，仅 LAN）

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/auth/status` | GET | 授权码状态 | `auth_pair.go` |
| `/api/auth/pair` | POST | 配对绑定 | `auth_pair.go` |
| `/api/auth/verify` | POST | Token 验证 | `auth_pair.go` |

### 配置管理（需 Token 认证）

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/config` | GET | 获取配置 | `config.go` |
| `/api/config` | PUT | 更新配置 | `config.go` |

### 对话功能（需 Token 认证）

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/chat` | POST | 同步对话 | `session.go` |
| `/api/chat/stream` | POST | 流式对话 | `session.go` |

### 模型管理

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/models` | GET | 模型列表 | `models.go` |
| `/api/models/default` | POST | 设置默认模型 | `models.go` |

### 技能管理

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/skills` | GET | 技能列表 | `skills.go` |
| `/api/skills` | POST | 安装技能 | `skills.go` |

### 网关控制

| 端点 | 方法 | 说明 | 文件 |
|------|------|------|------|
| `/api/gateway/status` | GET | 网关状态 | `gateway.go` |
| `/api/gateway/start` | POST | 启动网关 | `gateway.go` |
| `/api/gateway/stop` | POST | 停止网关 | `gateway.go` |
| `/api/gateway/events` | GET | SSE 事件流 | `events.go` |

## 局域网访问控制

`lan.go` 实现了 LAN 来源 IP 校验：

```go
// IsLANRequest 检查请求是否来自局域网
func IsLANRequest(r *http.Request) bool

// RequireLAN 中间件，限制仅 LAN 访问
func RequireLAN(next http.Handler) http.Handler
```

**允许的 IP 范围**:
- `127.0.0.0/8` (回环)
- `10.0.0.0/8` (Class A 私网)
- `172.16.0.0/12` (Class B 私网)
- `192.168.0.0/16` (Class C 私网)
- `169.254.0.0/16` (链路本地)
- `fc00::/7` (IPv6 私网)
- `fe80::/10` (IPv6 链路本地)

## 路由注册

`router.go` 负责注册所有 API 路由：

```go
func RegisterRoutes(router *mux.Router, services *Services)
```

## 认证方式

### Bearer Token

```bash
curl -H "Authorization: Bearer <token>" http://localhost:18800/api/config
```

### LAN Only

配对认证 API 仅允许局域网访问：

```bash
# 有效（来自 LAN）
curl http://192.168.1.100:18800/api/auth/status

# 无效（来自外部）
curl http://public-ip:18800/api/auth/status  # 403 Forbidden
```

## 测试

```bash
# 运行所有 API 测试
go test ./web/backend/api/... -v

# 运行特定测试
go test ./web/backend/api/... -v -run TestDiscovery
go test ./web/backend/api/... -v -run TestAuthPair
go test ./web/backend/api/... -v -run TestLanClient
```

## 相关文档

- [局域网发现实现](../../../docs/implementation/lan-discovery-status.md)
- [配对认证实现](../../../docs/implementation/lan-pairing-status.md)
- [Phase 1 总体文档](../../../../Cooking/moonhub-lan/phase1-mdns-pairing.md)
