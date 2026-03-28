# web/backend/api — HTTP API

MoonHub Web 后端 HTTP API 实现。路由由 [`router.go`](router.go) 中的 `Handler.RegisterRoutes` 注册到 `http.ServeMux`。

**相关文档**：[`web/README.md`](../README.md)（整体 Web 架构）、[`docs/implementation/lan-discovery-status.md`](../../docs/implementation/lan-discovery-status.md)、[`docs/implementation/lan-pairing-status.md`](../../docs/implementation/lan-pairing-status.md)。配套前端应用见独立仓库 **MoonHub-PWA**（与主仓库同工作区时常为 `MoonHub-PWA/`）。

## 文件结构（摘要）

| 文件 | 职责 |
| --- | --- |
| `router.go` | 汇总注册各子模块路由；`NewHandler(configPath, deviceStore, pairingManager)` |
| `discovery.go` | 单设备发现：`/api/ping`、`/api/system/info` |
| `discover.go` | 局域网 mDNS 扫描：`GET /api/discover`（LAN 限定） |
| `devices.go` | 已配对设备列表：`GET /api/devices`（LAN 限定） |
| `auth_pair.go` | 授权码配对与 Token 校验 |
| `lan.go` / `lan_client.go` | 局域网来源校验与客户端 IP 工具 |
| `channels.go` | 频道目录等（如 `GET /api/channels/catalog`） |
| `channels_crud.go` | 频道实例 CRUD + 状态（挂载在 `Handler` 上） |
| `config.go` | `GET` / `PUT` / `PATCH /api/config` |
| `session.go` | 对话与会话历史 |
| `gateway.go` / `events.go` | 网关生命周期与 SSE |
| `models.go` / `skills.go` / `tools.go` | 模型、技能、工具 API |
| `oauth.go` / `provisioning.go` / `startup.go` / … | 其余子系统 |

## 局域网访问控制

与发现、配对、设备列表、频道 CRUD 等相关的多个端点通过 `requireLANClient` / `lan.go` **仅允许来自私网或本机的请求**。允许的地址范围与实现细节见 [`lan.go`](lan.go)。

## API 端点分类

### 设备发现（无需 Token）

| 端点 | 方法 | 说明 | 文件 |
| --- | --- | --- | --- |
| `/api/ping` | GET | 设备在线检测 | `discovery.go` |
| `/api/system/info` | GET | 设备 / 系统信息 | `discovery.go` |

### 局域网发现与设备列表（无需 Token，LAN 限定）

| 端点 | 方法 | 说明 | 文件 |
| --- | --- | --- | --- |
| `/api/discover` | GET | 通过 mDNS 扫描局域网内 MoonHub 服务 | `discover.go` |
| `/api/devices` | GET | 返回已配对客户端列表 | `devices.go` |

### 配对与认证（无需 Token，LAN 限定）

| 端点 | 方法 | 说明 | 文件 |
| --- | --- | --- | --- |
| `/api/auth/status` | GET | 当前授权码状态 | `auth_pair.go` |
| `/api/auth/pair` | POST | 使用授权码配对，返回 Token | `auth_pair.go` |
| `/api/auth/verify` | POST | 校验 Token（`Authorization: Bearer` 或 JSON `{"token"}`） | `auth_pair.go` |

### 频道（LAN 限定；部分操作依赖配置读写）

| 端点 | 方法 | 说明 | 文件 |
| --- | --- | --- | --- |
| `/api/channels` | GET | 已配置频道实例列表（含状态摘要） | `channels_crud.go` |
| `/api/channels` | POST | 新增频道配置 | `channels_crud.go` |
| `/api/channels/{id}` | PATCH | 更新频道配置 | `channels_crud.go` |
| `/api/channels/{id}` | DELETE | 删除频道配置 | `channels_crud.go` |
| `/api/channels/{id}/status` | GET | 单个频道运行时状态 | `channels_crud.go` |

目录类接口（与具体实现对齐）见 `channels.go`（例如 `GET /api/channels/catalog`）。

### 配置与其它（需 Token 或按各 handler 约定）

| 区域 | 端点示例 | 文件 |
| --- | --- | --- |
| 配置 | `GET` / `PUT` / `PATCH /api/config` | `config.go` |
| 对话 | `POST /api/chat`、`POST /api/chat/stream` 等 | `session.go` |
| 模型 / 技能 / 工具 | `/api/models/*`、`/api/skills`、`/api/tools` 等 | `models.go`、`skills.go`、`tools.go` |
| 网关 | `/api/gateway/status`、`start`、`stop`、`/api/gateway/events` | `gateway.go`、`events.go` |

## 路由注册说明

`Handler.RegisterRoutes` 在内部依次调用 `registerConfigRoutes`、`registerGatewayRoutes`、`registerChannelCRUDRoutes` 等，并在末尾注册 `discovery`、`discover`、`devices`、`auth` 的 LAN 相关路由。配网相关路由在设置了 `SetProvisioningHandler` 时由 `provisioning` 子 handler 注册。

## 认证示例

### Bearer Token

```bash
curl -H "Authorization: Bearer <token>" http://127.0.0.1:18800/api/config
```

### 仅局域网可访问的端点

```bash
curl http://192.168.1.100:18800/api/auth/status
curl -X POST http://192.168.1.100:18800/api/auth/pair \
  -H "Content-Type: application/json" \
  -d '{"code":"AB1234"}'
```

## 测试

```bash
cd /path/to/MoonHub
go test ./web/backend/api/... -v
```

## 相关文档

- [局域网发现实现](../../docs/implementation/lan-discovery-status.md)
- [配对认证实现](../../docs/implementation/lan-pairing-status.md)
